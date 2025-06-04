package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"sort"
	"towerDefense/assets"

	"github.com/hajimehoshi/ebiten/v2"
)

// first tile ID constants for each tileset
const (
	groundFirstTileID = 1
	waterFirstTileID  = 257
)

// Constants for tile flipping
const (
	FlippedHorizontally = 0x80000000
	FlippedVertically   = 0x40000000
	FlippedDiagonally   = 0x20000000
	FlipMask            = FlippedHorizontally | FlippedVertically | FlippedDiagonally
)

// data we want for one layer in our list of layers
type TilemapLayerJSON struct {
	Data    []int               `json:"data"`
	Width   int                 `json:"width"`
	Height  int                 `json:"height"`
	Name    string              `json:"name"`
	Objects []TilemapObjectJSON `json:"objects,omitempty"`
	Type    string              `json:"type,omitempty"`
}

type TileImageMap struct {
	Images map[int]*ebiten.Image
}

// all layers in a tilemap
type TilemapJSON struct {
	Layers     []TilemapLayerJSON `json:"layers"`
	TileWidth  int                `json:"tilewidth"`
	TileHeight int                `json:"tileheight"`
}

// TilemapPropertyJSON defines the structure for properties within a Tiled object.
type TilemapPropertyJSON struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"` // Using 'any' (or interface{}) for flexibility
}

// Object definition for object layers (like waypoints)
type TilemapObjectJSON struct {
	Name       string                `json:"name"`
	Type       string                `json:"type"`
	X          float64               `json:"x"`
	Y          float64               `json:"y"`
	Properties []TilemapPropertyJSON `json:"properties"`
}

func (t TilemapJSON) LoadTiles() TileImageMap {
	tileMap := TileImageMap{
		Images: make(map[int]*ebiten.Image),
	}

	// collect all unique tile IDs from all layers
	uniqueTileIDs := make(map[int]bool)
	for _, layer := range t.Layers {
		for _, tileID := range layer.Data {
			if tileID != 0 { // 0 typically represents empty/no tile
				uniqueTileIDs[tileID] = true
			}
		}
	}

	// load image for each unique tile ID
	for tileID := range uniqueTileIDs {
		i, err := getTileImage(tileID)
		if err != nil {
			panic(err)
		}
		tileMap.Images[tileID] = i
	}

	return tileMap
}

// opens the file, parses it, and returns the json object + potential error
func NewTilemapJSON(filepath string) (*TilemapJSON, error) {
	contents, err := assets.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var tilemapJSON TilemapJSON
	err = json.Unmarshal(contents, &tilemapJSON)
	if err != nil {
		return nil, err
	}

	return &tilemapJSON, nil
}

// getTileImage returns the ebiten image for a given tile ID
func getTileImage(tileID int) (*ebiten.Image, error) {
	flippedH := (tileID & FlippedHorizontally) != 0
	flippedV := (tileID & FlippedVertically) != 0
	flippedD := (tileID & FlippedDiagonally) != 0

	// Get the actual tile ID without flip flags
	actualTileID := tileID &^ FlipMask // Use defined FlipMask

	var tilesetImage *ebiten.Image
	var localTileID int
	tileWidth, tileHeight := 64, 64 // Standard tile size
	var tilesPerRow int

	if actualTileID >= groundFirstTileID && actualTileID < waterFirstTileID {
		// Ground tileset
		tilesetImage = assets.GrassTileSet
		localTileID = actualTileID - groundFirstTileID // Use actualTileID, not tileID
		tilesPerRow = 16                               // Ground tileset has 16 columns
	} else if actualTileID >= waterFirstTileID { // Use actualTileID, not tileID
		// Water tileset
		tilesetImage = assets.WaterTileSet

		localTileID = actualTileID - waterFirstTileID // Use actualTileID, not tileID
		tilesPerRow = 70                              // Water tileset has 70 columns
	} else {
		return nil, nil // Invalid tile ID
	}

	// Calculate tile position in the tileset
	tileX := (localTileID % tilesPerRow) * tileWidth
	tileY := (localTileID / tilesPerRow) * tileHeight

	// Extract the tile from the tileset
	tileRect := image.Rect(tileX, tileY, tileX+tileWidth, tileY+tileHeight)
	tileImage := tilesetImage.SubImage(tileRect).(*ebiten.Image)

	// Apply flips if needed
	if flippedH || flippedV || flippedD {
		tileImage = applyFlips(tileImage, flippedH, flippedV, flippedD)
	}

	return tileImage, nil
}

func applyFlips(img *ebiten.Image, flippedH, flippedV, flippedD bool) *ebiten.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Create a new image for the flipped result
	flippedImg := ebiten.NewImage(w, h)

	opts := &ebiten.DrawImageOptions{}

	// Handle diagonal flip (90° rotation) first
	if flippedD {
		// 90° clockwise rotation + horizontal flip = diagonal flip in Tiled
		opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
		opts.GeoM.Rotate(3.14159 / 2) // 90 degrees in radians
		opts.GeoM.Scale(-1, 1)        // Horizontal flip
		opts.GeoM.Translate(float64(h)/2, float64(w)/2)
	} else {
		// Handle horizontal flip
		if flippedH {
			opts.GeoM.Scale(-1, 1)
			opts.GeoM.Translate(float64(w), 0)
		}

		// Handle vertical flip
		if flippedV {
			opts.GeoM.Scale(1, -1)
			opts.GeoM.Translate(0, float64(h))
		}
	}

	flippedImg.DrawImage(img, opts)
	return flippedImg
}

// GetWaypoints returns a slice of waypoints from the waypoints layer, sorted numerically by name
func (t *TilemapJSON) GetWaypoints() []PathNode {
	waypoints := []struct {
		Index int
		Node  PathNode
	}{}

	for _, layer := range t.Layers {
		if layer.Type == "objectgroup" && layer.Name == "waypoints" {
			for _, obj := range layer.Objects {
				// Try to parse the name as an integer
				var idx int
				_, err := fmt.Sscanf(obj.Name, "%d", &idx)
				if err != nil {
					continue // skip if not a number
				}
				tileX := int(obj.X) / t.TileWidth
				tileY := int(obj.Y) / t.TileHeight
				waypoints = append(waypoints, struct {
					Index int
					Node  PathNode
				}{idx, PathNode{X: tileX, Y: tileY}})
			}
			break
		}
	}

	// Sort by Index
	sort.Slice(waypoints, func(i, j int) bool {
		return waypoints[i].Index < waypoints[j].Index
	})

	// Extract just the PathNodes
	result := make([]PathNode, len(waypoints))
	for i, wp := range waypoints {
		result[i] = wp.Node
	}
	return result
}
