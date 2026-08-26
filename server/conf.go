package server

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/internal/packbuilder"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
	"github.com/df-mc/dragonfly/server/player/playerdb"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/biome"
	"github.com/df-mc/dragonfly/server/world/generator"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// Config contains options for starting a Minecraft server.
type Config struct {
	Log *slog.Logger
	Listeners []func(conf Config) (Listener, error)
	Name string
	Resources []*resource.Pack
	ResourcesRequired bool
	DisableResourceBuilding bool
	Allower Allower
	AuthDisabled bool
	MuteEmoteChat bool
	MaxPlayers int
	MaxChunkRadius int
	JoinMessage, QuitMessage, ShutdownMessage chat.Translation
	StatusProvider minecraft.ServerStatusProvider
	Compression packet.Compression
	AcceptedProtocols []minecraft.Protocol
	AcceptedProtocolsProvider func(blocks world.BlockRegistry) ([]minecraft.Protocol, error)
	PlayerProvider player.Provider
	WorldProvider world.Provider
	ReadOnlyWorld bool
	Generator func(dim world.Dimension) world.Generator
	RandomTickSpeed int
	SaveInterval time.Duration
	ChunkUnloadInterval time.Duration
	ChunkLoadWorkers int
	Entities world.EntityRegistry
	Blocks world.BlockRegistry
	DisableNether bool
	DisableEnd bool
	DisableFireTick bool
	DisableHopperTick bool
	DisableRedstoneTick bool
	DisableSubChunkRequests bool
	DisableVanillaRecipes bool
	RandomTickFilter func(world.Block) bool
	// Minigame contains opt-in fixed-map/minigame hot-path optimisations. The
	// zero value preserves regular Dragonfly behaviour.
	Minigame world.MinigameConfig
	OperatorsFile string
}

// New creates a Server using fields of conf. The Server's worlds are created
// and connections from the Server's listeners may be accepted by calling
// Server.Listen() and Server.Accept() afterwards.
func (conf Config) New() *Server {
	if conf.Log == nil {
		conf.Log = slog.Default()
	}
	if len(conf.Listeners) == 0 {
		conf.Log.Warn("config: no listeners set, no connections will be accepted")
	}
	if conf.Name == "" {
		conf.Name = "Dragonfly Server"
	}
	if conf.StatusProvider == nil {
		conf.StatusProvider = statusProvider{name: conf.Name}
	}
	if conf.PlayerProvider == nil {
		conf.PlayerProvider = player.NopProvider{}
	}
	if conf.Allower == nil {
		conf.Allower = allower{}
	}
	if conf.WorldProvider == nil {
		conf.WorldProvider = world.NopProvider{}
	}
	if conf.Generator == nil {
		conf.Generator = loadGenerator
	}
	if conf.MaxChunkRadius == 0 {
		conf.MaxChunkRadius = 12
	}
	if conf.ShutdownMessage.Zero() {
		conf.ShutdownMessage = chat.MessageServerDisconnect
	}
	if len(conf.Entities.Types()) == 0 {
		conf.Entities = entity.DefaultRegistry
	}
	if conf.Blocks == nil {
		conf.Blocks = world.DefaultBlockRegistry
	}

	conf.Blocks.Finalize()
	world.DefaultBlockRegistry.Finalize()
	if conf.AcceptedProtocolsProvider != nil {
		protocols, err := conf.AcceptedProtocolsProvider(conf.Blocks)
		if err != nil {
			panic(fmt.Sprintf("config: build accepted protocols: %v", err))
		}
		conf.AcceptedProtocols = append(conf.AcceptedProtocols, protocols...)
	}

	if !conf.DisableResourceBuilding {
		if pack, ok := packbuilder.BuildResourcePack(conf.Blocks); ok {
			conf.Resources = append(conf.Resources, pack)
		}
	}
	conf.Resources = slices.Clone(conf.Resources)
	conf.AcceptedProtocols = slices.Clone(conf.AcceptedProtocols)

	if conf.OperatorsFile == "" {
		conf.OperatorsFile = "ops.txt"
	}
	srv := &Server{
		conf:     conf,
		incoming: make(chan incoming),
		p:        make(map[uuid.UUID]*onlinePlayer),
		ops:      loadOperators(conf.OperatorsFile),
		world:    &world.World{}, nether: &world.World{}, end: &world.World{},
	}
	for _, lf := range conf.Listeners {
		l, err := lf(conf)
		if err != nil {
			conf.Log.Error("create listener: " + err.Error())
			continue
		}
		srv.listeners = append(srv.listeners, l)
	}

	creative_registerCreativeItems()
	if !conf.DisableVanillaRecipes {
		recipe_registerVanilla()
	}

	srv.world = srv.createWorld(world.Overworld, &srv.nether, &srv.end)
	srv.world.SetMinigameConfig(conf.Minigame)
	if !conf.DisableNether {
		srv.nether = srv.createWorld(world.Nether, &srv.world, &srv.end)
		srv.nether.SetMinigameConfig(conf.Minigame)
	} else {
		srv.nether = nil
	}
	if !conf.DisableEnd {
		srv.end = srv.createWorld(world.End, &srv.nether, &srv.world)
		srv.end.SetMinigameConfig(conf.Minigame)
	} else {
		srv.end = nil
	}

	return srv
}

// UserConfig is the user configuration for a Dragonfly server. It holds
// settings that affect different aspects of the server, such as its name and
// maximum players. UserConfig may be serialised and can be converted to a
// Config by calling UserConfig.Config().
type UserConfig struct {
	Network struct {
		Address string
	}
	Server struct {
		Name string
		AuthEnabled bool
		DisableJoinQuitMessages bool
		MuteEmoteChat bool
		OperatorsFile string
	}
	World struct {
		SaveData bool
		Folder string
		DisableNether bool
		DisableEnd bool
		DisableFireTick bool
		DisableHopperTick bool
		DisableRedstoneTick bool
		DisableSubChunkRequests bool
		RandomTickSpeed int
		DisableVanillaRecipes bool
		Minigame struct {
			DisablePlayerSurvivalTicks bool
			DisablePlayerEffectTicks bool
			DisablePortalTicks bool
			DeduplicatePlayerCollisionTicks bool
			FastSetBlock bool
			FastBreakBlock bool
			DisableBlockTicks bool
			DisableScheduledBlockTicks bool
			ActiveEntityTicking bool
			MovementDirtyChunkTracking bool
		}
	}
	Players struct {
		MaxCount int
		MaximumChunkRadius int
		SaveData bool
		Folder string
	}
	Resources struct {
		AutoBuildPack bool
		Folder string
		Required bool
	}
}

// Config converts a UserConfig to a Config, so that it may be used for creating
// a Server. An error is returned if creating data providers or loading
// resources failed.
func (uc UserConfig) Config(log *slog.Logger) (Config, error) {
	var err error
	conf := Config{
		Log:                     log,
		Name:                    uc.Server.Name,
		ResourcesRequired:       uc.Resources.Required,
		AuthDisabled:            !uc.Server.AuthEnabled,
		MuteEmoteChat:           uc.Server.MuteEmoteChat,
		MaxPlayers:              uc.Players.MaxCount,
		MaxChunkRadius:          uc.Players.MaximumChunkRadius,
		DisableResourceBuilding: !uc.Resources.AutoBuildPack,
		DisableNether:           uc.World.DisableNether,
		DisableEnd:              uc.World.DisableEnd,
		DisableFireTick:         uc.World.DisableFireTick,
		DisableHopperTick:       uc.World.DisableHopperTick,
		DisableRedstoneTick:     uc.World.DisableRedstoneTick,
		DisableSubChunkRequests: uc.World.DisableSubChunkRequests,
		RandomTickSpeed:         uc.World.RandomTickSpeed,
		DisableVanillaRecipes:   uc.World.DisableVanillaRecipes,
		Minigame: world.MinigameConfig{
			DisablePlayerSurvivalTicks:     uc.World.Minigame.DisablePlayerSurvivalTicks,
			DisablePlayerEffectTicks:       uc.World.Minigame.DisablePlayerEffectTicks,
			DisablePortalTicks:             uc.World.Minigame.DisablePortalTicks,
			DeduplicatePlayerCollisionTicks: uc.World.Minigame.DeduplicatePlayerCollisionTicks,
			FastSetBlock:                   uc.World.Minigame.FastSetBlock,
			FastBreakBlock:                 uc.World.Minigame.FastBreakBlock,
			DisableBlockTicks:              uc.World.Minigame.DisableBlockTicks,
			DisableScheduledBlockTicks:     uc.World.Minigame.DisableScheduledBlockTicks,
			ActiveEntityTicking:            uc.World.Minigame.ActiveEntityTicking,
			MovementDirtyChunkTracking:     uc.World.Minigame.MovementDirtyChunkTracking,
		},
		OperatorsFile: uc.Server.OperatorsFile,
	}
	if !uc.Server.DisableJoinQuitMessages {
		conf.JoinMessage, conf.QuitMessage = chat.MessageJoin, chat.MessageQuit
	}
	if uc.World.SaveData {
		conf.WorldProvider, err = mcdb.Config{Log: log}.Open(uc.World.Folder)
		if err != nil {
			return conf, fmt.Errorf("create world provider: %w", err)
		}
	}
	conf.Resources, err = loadResources(uc.Resources.Folder)
	if err != nil {
		return conf, fmt.Errorf("load resources: %w", err)
	}
	if uc.Players.SaveData {
		conf.PlayerProvider, err = playerdb.NewProvider(uc.Players.Folder)
		if err != nil {
			return conf, fmt.Errorf("create player provider: %w", err)
		}
	}
	conf.Listeners = append(conf.Listeners, uc.listenerFunc)
	return conf, nil
}

// loadResources loads all resource packs found in a directory passed.
func loadResources(dir string) ([]*resource.Pack, error) {
	_ = os.MkdirAll(dir, 0777)

	resources, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	packs := make([]*resource.Pack, len(resources))
	for i, entry := range resources {
		packs[i], err = resource.ReadPath(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("compile resource (%v): %w", entry.Name(), err)
		}
	}
	return packs, nil
}

// loadGenerator loads a standard world.Generator for a world.Dimension.
func loadGenerator(dim world.Dimension) world.Generator {
	switch dim {
	case world.Overworld:
		return generator.NewFlat(biome.Plains{}, []world.Block{block.Grass{}, block.Dirt{}, block.Dirt{}, block.Bedrock{}})
	case world.Nether:
		return generator.NewFlat(biome.NetherWastes{}, []world.Block{block.Netherrack{}, block.Netherrack{}, block.Netherrack{}, block.Bedrock{}})
	case world.End:
		return generator.NewFlat(biome.End{}, []world.Block{block.EndStone{}, block.EndStone{}, block.EndStone{}, block.Bedrock{}})
	}
	panic("should never happen")
}

// DefaultConfig returns a configuration with the default values filled out.
func DefaultConfig() UserConfig {
	c := UserConfig{}
	c.Network.Address = ":19132"
	c.Server.Name = "Dragonfly Server"
	c.Server.AuthEnabled = true
	c.World.SaveData = true
	c.World.Folder = "world"
	c.Players.MaximumChunkRadius = 32
	c.Players.SaveData = true
	c.Players.Folder = "players"
	c.Resources.AutoBuildPack = true
	c.Resources.Folder = "resources"
	c.Resources.Required = false
	return c
}

// noinspection ALL
//
//go:linkname creative_registerCreativeItems github.com/df-mc/dragonfly/server/item/creative.registerCreativeItems
func creative_registerCreativeItems()

// noinspection ALL
//
//go:linkname recipe_registerVanilla github.com/df-mc/dragonfly/server/item/recipe.registerVanilla
func recipe_registerVanilla()
