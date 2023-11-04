package game

import (
	"fmt"
	"testing"
)

// func TestUpdate(t *testing.T) {
// 	g := &Game{}
// 	g.InitializeState()
// 	g.InitializeBoard()
// 	g.isPaused = false

// 	for i := 1; i <= 1000; i++ {
// 		g.Update()
// 		err := verifyNeighbourCounts(g.gridX, g.gridY, g.worldGrid)
// 		if err != nil {
// 			t.Fatalf("failed board verification: %v", err)
// 		}
// 	}
// }

func BenchmarkUpdate(b *testing.B) {

	for i := 0; i < 7; i++ {
		POOL_SIZE = 1 << i
		g := &Game{}
		g.InitializeState()
		g.InitializeBoard()
		g.isPaused = false
		b.Run(fmt.Sprintf("%4d", POOL_SIZE), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				g.updateBoard()
			}
		})
		close(g.taskChannel)
	}
}

func BenchmarkUpdateAlt(b *testing.B) {
	for i := 0; i < 7; i++ {
		POOL_SIZE = 1 << i
		g := &Game{}
		g.InitializeState()
		g.InitializeBoard()
		g.isPaused = false
		b.Run(fmt.Sprintf("%4d", POOL_SIZE), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				g.updateBoardAlt()
			}
			g.InitializeState()
		})
		close(g.taskChannel)
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
