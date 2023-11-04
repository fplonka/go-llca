package game

// For comparing performance in benchmarks
func (g *Game) updateBoardAlt() error {
	copy(g.buffer, g.worldGrid)

	// Divide the board into equal-sized parts and create tasks for each part.
	numParts := POOL_SIZE
	if numParts > g.gridY/4 { // Cap the number of parts on small boards.
		numParts = g.gridY / 4
	}
	rowsPerPart := g.gridY / numParts
	for i := 0; i < numParts; i++ {
		minY := 1 + i*rowsPerPart
		maxY := minY + rowsPerPart - 1
		if i == numParts-1 {
			maxY = g.gridY
		}

		g.wg.Add(1)
		// We can't update the border regions of a part since that would lead to data races.
		g.taskChannel <- Task{minY: minY + 1, maxY: maxY - 1}
	}
	g.wg.Wait()

	// Update the border regions now that it's safe to do so.
	g.wg.Add(2)
	g.taskChannel <- Task{minY: 1, maxY: 1}
	g.taskChannel <- Task{minY: g.gridY, maxY: g.gridY}
	for i := 1; i < numParts; i++ {
		minY := 1 + i*rowsPerPart

		g.wg.Add(1)
		g.taskChannel <- Task{minY: minY - 1, maxY: minY}
	}
	g.wg.Wait()

	copy(g.worldGrid, g.buffer)

	return nil
}

func (g *Game) updateRangeAlt(minY, maxY int) {
	// Update the game board.
	// We do this more efficiently by copying the board state into a buffer and modifying only those cells in the
	// buffer which are changing state (becoming alive or dying).
	for i := minY; i <= maxY; i++ {
		for j := 1; j <= g.gridX; j++ {
			// Getting the "2D g.worldGrid[i][j]" index from the 1D slice. +2 because of the board edge border.
			val := g.worldGrid[i*(g.gridX+2)+j]
			gridXPlusTwo := g.gridX + 2

			if g.becomesAliveTable[val] { // Checking if the cell is becoming alive. val&1 == 0 ensures that
				// this cell was dead previously, and val>>1 gets the number of live neighbours.

				// g.buffer[ind] |= 1 // Set the last bit to 1 to indicate that this cell is now alive.
				g.buffer[(i-1)*(gridXPlusTwo)+j-1] += 2
				g.buffer[(i-1)*(gridXPlusTwo)+j] += 2
				g.buffer[(i-1)*(gridXPlusTwo)+j+1] += 2
				g.buffer[(i)*(gridXPlusTwo)+j-1] += 2
				g.buffer[(i)*(gridXPlusTwo)+j] += 1
				g.buffer[(i)*(gridXPlusTwo)+j+1] += 2
				g.buffer[(i+1)*(gridXPlusTwo)+j-1] += 2
				g.buffer[(i+1)*(gridXPlusTwo)+j] += 2
				g.buffer[(i+1)*(gridXPlusTwo)+j+1] += 2
				// -1 because i and j and 1-indexed due to the border, which the game board image doesn't have.
				setPixel(g.pixels, g.gridX, j-1, i-1, 0)

			} else if g.becomesDeadTable[val] { // Checking if the cell is becoming dead. val&1 == 1 ensures
				// that this cell was alive previously. Since this cell is alive, val>>1 is the one more than the number
				// of live neighbours, as this cell is also counted in val>1, so we check val>>1-1 in SRules.

				// The rest of this case is analogous to the cell becoming alive case.
				// g.buffer[ind] -= 1 // Set the last bit to 0 to indicate that this cell is now dead.
				g.buffer[(i-1)*(gridXPlusTwo)+j-1] -= 2
				g.buffer[(i-1)*(gridXPlusTwo)+j] -= 2
				g.buffer[(i-1)*(gridXPlusTwo)+j+1] -= 2
				g.buffer[(i)*(gridXPlusTwo)+j-1] -= 2
				g.buffer[(i)*(gridXPlusTwo)+j] -= 1
				g.buffer[(i)*(gridXPlusTwo)+j+1] -= 2
				g.buffer[(i+1)*(gridXPlusTwo)+j-1] -= 2
				g.buffer[(i+1)*(gridXPlusTwo)+j] -= 2
				g.buffer[(i+1)*(gridXPlusTwo)+j+1] -= 2
				setPixel(g.pixels, g.gridX, j-1, i-1, 1)
			}
		}
	}
}
