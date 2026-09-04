package player

import "testing"

func TestMetadataSettersIgnoreUnchangedValues(t *testing.T) {
	t.Run("name tag", func(t *testing.T) {
		p := &Player{playerData: &playerData{nameTag: "same"}}
		p.SetNameTag("same")
	})

	t.Run("score tag", func(t *testing.T) {
		p := &Player{playerData: &playerData{scoreTag: "same"}}
		p.SetScoreTag("same")
	})

	t.Run("always show name tag", func(t *testing.T) {
		p := &Player{playerData: &playerData{alwaysShowNameTag: true}}
		p.SetAlwaysShowNameTag(true)
	})
}
