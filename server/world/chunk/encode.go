package chunk

import (
	"bytes"
	"sync"
)

const (
	// SubChunkVersion is the current version of the written sub chunks, specifying the format they are
	// written on disk and over network.
	SubChunkVersion = 9
	// CurrentBlockVersion is the current version of blocks (states) of the game. This version is composed
	// of 4 bytes indicating a version, interpreted as a big endian int. The current version represents
	// 1.16.0.14 {1, 16, 0, 14}.
	CurrentBlockVersion int32 = 18040335
)

var (
	// pool is used to pool byte buffers used for encoding chunks.
	pool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}
)

type (
	// SerialisedData holds the serialised data of a chunk. It consists of the chunk's block data itself, a height
	// map, the biomes and entities and block entities.
	SerialisedData struct {
		// sub holds the data of the serialised sub chunks in a chunk. Sub chunks that are empty or that otherwise
		// don't exist are represented as an empty slice (or technically, nil).
		SubChunks [][]byte
		// Biomes is the biome data of the chunk, which is composed of a biome storage for each sub-chunk.
		Biomes []byte
	}
	// blockEntry represents a block as found in a disk save of a world.
	blockEntry struct {
		Name    string         `nbt:"name"`
		State   map[string]any `nbt:"states"`
		Version int32          `nbt:"version"`
	}
)

// Encode encodes Chunk to an intermediate representation SerialisedData. An Encoding may be passed to encode either for
// network or disk purposed, the most notable difference being that the network encoding generally uses varints and no
// NBT.
func Encode(c *Chunk, e Encoding) SerialisedData {
	if provider, ok := e.(interface {
		networkChunkFormat() (NetworkChunkFormatController, bool)
	}); ok {
		if format, configured := provider.networkChunkFormat(); configured {
			return encodeConfiguredNetworkChunk(c, e, format)
		}
	}
	d := SerialisedData{SubChunks: make([][]byte, len(c.sub))}
	for i := range c.sub {
		d.SubChunks[i] = EncodeSubChunk(c, e, i)
	}
	d.Biomes = EncodeBiomes(c, e)
	return d
}

// EncodeSubChunk encodes a sub-chunk from a chunk into bytes. An Encoding may be passed to encode either for network or
// disk purposed, the most notable difference being that the network encoding generally uses varints and no NBT.
func EncodeSubChunk(c *Chunk, e Encoding, ind int) []byte {
	return encodeSubChunk(c, e, ind, SubChunkVersion, true)
}

func encodeSubChunk(c *Chunk, e Encoding, ind int, version byte, includeIndex bool) []byte {
	buf := pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		pool.Put(buf)
	}()

	s := c.sub[ind]
	_, _ = buf.Write([]byte{version, byte(len(s.storages))})
	if includeIndex {
		_ = buf.WriteByte(uint8(ind + (c.r[0] >> 4)))
	}
	for _, storage := range s.storages {
		encodePalettedStorage(buf, storage, nil, e, BlockPaletteEncoding{Blocks: c.br})
	}
	sub := make([]byte, buf.Len())
	_, _ = buf.Read(sub)
	return sub
}

func encodeConfiguredNetworkChunk(c *Chunk, e Encoding, format NetworkChunkFormatController) SerialisedData {
	minY, maxY := format.NetworkChunkRange()
	if minY > maxY || minY&15 != 0 || maxY&15 != 15 {
		panic("chunk: network chunk range must be 16-block aligned")
	}
	start := int((minY - int16(c.r[0])) >> 4)
	end := int((maxY - int16(c.r[0])) >> 4)
	if start < 0 || end >= len(c.sub) {
		panic("chunk: network chunk range falls outside the live chunk")
	}
	highest := -1
	for index := end; index >= start; index-- {
		if !c.sub[index].Empty() {
			highest = index
			break
		}
	}
	data := SerialisedData{}
	if highest >= start {
		data.SubChunks = make([][]byte, highest-start+1)
		version := format.NetworkSubChunkVersion()
		for index := start; index <= highest; index++ {
			if c.sub[index].Empty() {
				data.SubChunks[index-start] = []byte{version, 0}
				continue
			}
			data.SubChunks[index-start] = encodeSubChunk(c, e, index, version, version >= 9)
		}
	}
	if format.NetworkBiomes2D() {
		data.Biomes = encodeBiomes2D(c, minY, maxY)
	} else {
		data.Biomes = EncodeBiomes(c, e)
	}
	return data
}

func encodeBiomes2D(c *Chunk, minY, maxY int16) []byte {
	biomes := make([]byte, 256)
	for x := uint8(0); x < 16; x++ {
		for z := uint8(0); z < 16; z++ {
			y := minY
			for candidate := maxY; candidate >= minY; candidate-- {
				if c.Block(x, candidate, z, 0) != c.air {
					y = candidate
					break
				}
			}
			biomes[int(z)<<4|int(x)] = byte(c.Biome(x, y, z))
		}
	}
	return biomes
}

// EncodeBiomes encodes the biomes of a chunk into bytes. An Encoding may be passed to encode either for network or
// disk purposed, the most notable difference being that the network encoding generally uses varints and no NBT.
func EncodeBiomes(c *Chunk, e Encoding) []byte {
	buf := pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		pool.Put(buf)
	}()

	var previous *PalettedStorage
	reuse := true
	if controller, ok := e.(interface{ canReuseBiomePalettes() bool }); ok {
		reuse = controller.canReuseBiomePalettes()
	}
	for _, b := range c.biomes {
		var reusable *PalettedStorage
		if reuse {
			reusable = previous
		}
		encodePalettedStorage(buf, b, reusable, e, BiomePaletteEncoding)
		previous = b
	}
	biomes := make([]byte, buf.Len())
	_, _ = buf.Read(biomes)
	return biomes
}

// encodePalettedStorage encodes a PalettedStorage into a bytes.Buffer. The Encoding passed is used to write the Palette
// of the PalettedStorage.
func encodePalettedStorage(buf *bytes.Buffer, storage, previous *PalettedStorage, e Encoding, pe paletteEncoding) {
	if mapper, ok := e.(interface {
		mapStorage(*PalettedStorage, paletteEncoding) *PalettedStorage
	}); ok {
		storage = mapper.mapStorage(storage, pe)
	}
	if storage.Equal(previous) {
		_, _ = buf.Write([]byte{0x7f<<1 | e.network()})
		return
	}
	b := make([]byte, len(storage.indices)*4+1)
	b[0] = byte(storage.bitsPerIndex<<1) | e.network()

	for i, v := range storage.indices {
		// Explicitly don't use the binary package to greatly improve performance of writing the uint32s.
		b[i*4+1], b[i*4+2], b[i*4+3], b[i*4+4] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	_, _ = buf.Write(b)

	e.encodePalette(buf, storage.palette, pe)
}
