package turtlesoup

import "testing"

func TestPuzzleLoaderLoadAndRandom(t *testing.T) {
	loader, err := NewPuzzleLoader()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := loader.All()
	if len(all) == 0 {
		t.Fatalf("expected puzzles to be loaded")
	}

	puzzle, err := loader.GetRandomPuzzleByDifficulty(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if puzzle.Difficulty != 1 {
		t.Fatalf("expected difficulty 1, got %d", puzzle.Difficulty)
	}
}
