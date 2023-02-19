package main

import (
	"fmt"
	"testing"
)

func TestUpdate(t *testing.T) {
	g := &Game{}
	g.initializeState() // Only called here.
	g.initializeBoard()
	g.isPaused = false

	for i := 1; i <= 1000; i++ {
		g.Update()
		err := verifyNeighbourCounts(g.gridX, g.gridY, g.worldGrid)
		if err != nil {
			t.Fatalf("failed board verification: %v", err)
		}
	}
}

func verifyNeighbourCounts(gridX, gridY int, worldGrid []int8) error {
	for i := 1; i <= gridY; i++ {
		for j := 1; j <= gridY; j++ {
			desiredVal := int8(0)
			for a := -1; a <= 1; a++ {
				for b := -1; b <= 1; b++ {
					desiredVal += 2 * (worldGrid[(i+a)*(gridX+2)+j+b] & 1)
				}
			}
			desiredVal |= (worldGrid[(i)*(gridX+2)+j] & 1)
			ind := i*(gridX+2) + j
			val := worldGrid[ind]
			if desiredVal != val {
				return fmt.Errorf("incorrect at (%v %v), should be %v but is %v", j, i, desiredVal, val)
			}

		}
	}

	return nil
}
