package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/exp/slices"
	"os"
	"strings"
)

// todo: resizing
// todo: win condition

const gridWidth = 8
const gridHeight = 8

type vector2d struct {
	x int
	y int
}

type player int

const (
	DarkPlayer player = iota
	LightPlayer
	Blank = -1
)

func (p player) String() string {
	return [...]string{"Dark Player", "Light Player"}[p]
}

func (p player) toSymbol() string {
	return [...]string{"X", "O"}[p]
}

type grid [gridHeight][gridWidth]player

type view int

const (
	PointSelection view = iota
	PointConfirmation
)

type model struct {
	grid          grid
	selectedPoint vector2d
	view          view
	currentPlayer player
	disksFlipped  []vector2d
}

func newGrid() *grid {
	var g grid

	for i := 0; i < gridHeight; i++ {
		for j := 0; j < gridWidth; j++ {
			g[i][j] = Blank
		}
	}

	g[3][3] = LightPlayer
	g[4][4] = LightPlayer
	g[3][4] = DarkPlayer
	g[4][3] = DarkPlayer

	return &g
}

func initialModel() model {
	return model{
		grid:          *newGrid(),
		selectedPoint: vector2d{3, 3},
		view:          PointSelection,
		currentPlayer: DarkPlayer,
		disksFlipped:  make([]vector2d, 0),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.view {
	case PointSelection:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "w":
				m.selectedPoint.y--
				m.selectedPoint.y = (m.selectedPoint.y + gridHeight) % gridHeight
			case "down", "s":
				m.selectedPoint.y++
				m.selectedPoint.y = (m.selectedPoint.y + gridHeight) % gridHeight
			case "left", "a":
				m.selectedPoint.x--
				m.selectedPoint.x = (m.selectedPoint.x + gridWidth) % gridWidth
			case "right", "d":
				m.selectedPoint.x++
				m.selectedPoint.x = (m.selectedPoint.x + gridWidth) % gridWidth
			case "enter", " ":
				availablePoints := getAvailablePoints(m.grid)

				if slices.Contains(availablePoints, m.selectedPoint) {
					m.grid[m.selectedPoint.y][m.selectedPoint.x] = m.currentPlayer

					flip(&m)

					m.view = PointConfirmation
				}
			}
		}
	case PointConfirmation:
		switch msg.(type) {
		case tea.KeyMsg:
			m.view = PointSelection
		}

		// Update current player *after* displaying PointConfirmation view
		if m.currentPlayer == DarkPlayer {
			m.currentPlayer = LightPlayer
		} else if m.currentPlayer == LightPlayer {
			m.currentPlayer = DarkPlayer
		}
	}

	return m, nil
}

func getAvailablePoints(g grid) []vector2d {
	// Get all non-blank points in grid
	nonBlankPoints := make([]vector2d, 0)
	for i, row := range g {
		for j, cell := range row {
			if cell != Blank {
				nonBlankPoints = append(nonBlankPoints, vector2d{j, i})
			}
		}
	}

	// Get all neighbours of non-blank points in grid
	neighbors := make([]vector2d, 0, len(nonBlankPoints)*8)
	for _, nonBlankPoint := range nonBlankPoints {
		for i := -1; i <= 1; i++ {
			for j := -1; j <= 1; j++ {
				if i != 0 || j != 0 {
					neighbor := vector2d{nonBlankPoint.x + j, nonBlankPoint.y + i}
					neighbors = append(neighbors, neighbor)
				}
			}
		}
	}

	// Keep only neighbours that are blank and inside the grid
	neighbors1 := make([]vector2d, 0, len(neighbors)) // todo: rename
	for _, neighbor := range neighbors {
		if isPointInsideGrid(neighbor) && g[neighbor.y][neighbor.x] == Blank {
			neighbors1 = append(neighbors1, neighbor)
		}
	}

	return neighbors1
}

// so confused lol
//func getNextAvailablePoint(availablePoints []vector2d, selectedPoint vector2d, direction vector2d) vector2d {
//	availablePointsInDirection := make([]vector2d, 0, len(availablePoints))
//	for _, p := range availablePoints {
//		if (direction.x != 0 || p.x == selectedPoint.x) && (direction.y != 0 || p.y == selectedPoint.y) {
//			availablePointsInDirection = append(availablePointsInDirection, p)
//		}
//	}
//
//	if len(availablePointsInDirection) == 0 {
//		return selectedPoint
//	}
//
//	sortedAvailablePointsInDirection := make([]vector2d, len(availablePointsInDirection))
//	copy(sortedAvailablePointsInDirection, availablePointsInDirection)
//	sort.Slice(sortedAvailablePointsInDirection, func(i, j int) bool {
//		point1 := sortedAvailablePointsInDirection[i]
//		point2 := sortedAvailablePointsInDirection[j]
//
//		if direction.x == -1 {
//			if point1.x < selectedPoint.x {
//				point1.x += gridWidth
//			}
//			if point2.x < selectedPoint.x {
//				point2.x += gridWidth
//			}
//
//			return point2.x > point2.x
//		} else if direction.x == 1 {
//			if point1.x > selectedPoint.x {
//				point1.x += gridWidth
//			}
//			if point2.x > selectedPoint.x {
//				point2.x += gridWidth
//			}
//
//			return point2.x > point2.x
//		}
//	})
//
//	return sortedAvailablePointsInDirection[0]
//}

func isPointInsideGrid(p vector2d) bool {
	return p.x >= 0 && p.x < gridWidth && p.y >= 0 && p.y < gridHeight
}

func flip(m *model) {
	// Maybe generate these automatically
	directions := []vector2d{
		{0, 1},
		{1, 0},
		{1, 1},
		{0, -1},
		{-1, 0},
		{-1, -1},
		{1, -1},
		{-1, 1},
	}

	m.disksFlipped = make([]vector2d, 0, 10)
	for _, d := range directions {
		currentPoint := m.selectedPoint
		isInsideGrid := isPointInsideGrid(currentPoint)
		isNotBlank := m.grid[currentPoint.y][currentPoint.x] != Blank
		isCurrentPlayer := false
		pointsToFlip := make([]vector2d, 0)
		for isInsideGrid && isNotBlank && !isCurrentPlayer {
			currentPoint = vector2d{x: currentPoint.x + d.x, y: currentPoint.y + d.y}

			isInsideGrid = isPointInsideGrid(currentPoint)
			if !isInsideGrid {
				break
			}

			isNotBlank = m.grid[currentPoint.y][currentPoint.x] != Blank
			isCurrentPlayer = m.grid[currentPoint.y][currentPoint.x] == m.currentPlayer

			if isInsideGrid && isNotBlank && !isCurrentPlayer {
				pointsToFlip = append(pointsToFlip, currentPoint)
			}
		}

		// If disk of current player's colour is reached, change all the intermediate disks to the current player's colour
		// If blank cell or edge of grid is reached, don't change any disks
		if isCurrentPlayer {
			for _, p := range pointsToFlip {
				// Flip disk
				m.grid[p.y][p.x] = m.currentPlayer
			}

			m.disksFlipped = append(m.disksFlipped, pointsToFlip...)
		}
	}
}

var darkPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#ffffff")).
	Background(lipgloss.Color("#000000"))

var lightPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(lipgloss.Color("#ffffff"))

var selectedDarkPlayerStyle = lipgloss.NewStyle().
	Blink(true).
	Underline(true).
	Bold(true).
	Foreground(lipgloss.Color("105")).
	Background(lipgloss.Color("#000000"))

var selectedLightPlayerStyle = lipgloss.NewStyle().
	Blink(true).
	Foreground(lipgloss.Color("#000000")).
	Background(lipgloss.Color("105"))

var selectedBlankStyle = lipgloss.NewStyle().
	Blink(true).
	Background(lipgloss.Color("105"))

var availablePointStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#212121"))

func computeScores(g grid) map[player]int {
	m := make(map[player]int)
	for _, row := range g {
		for _, cell := range row {
			if cell != Blank {
				m[cell]++
			}
		}
	}
	return m
}

func (m model) View() string {
	var gridStringBuilder strings.Builder
	availablePoints := getAvailablePoints(m.grid)
	scores := computeScores(m.grid)

	for i, row := range m.grid {
		for j, cell := range row {
			point := vector2d{j, i}
			if (m.view == PointSelection && point == m.selectedPoint) ||
				(m.view == PointConfirmation && slices.Contains(m.disksFlipped, point)) {
				switch cell {
				case DarkPlayer:
					gridStringBuilder.WriteString(selectedDarkPlayerStyle.Render("X"))
				case LightPlayer:
					gridStringBuilder.WriteString(selectedLightPlayerStyle.Render("O"))
				default:
					gridStringBuilder.WriteString(selectedBlankStyle.Render(" "))
				}
			} else {
				switch cell {
				case DarkPlayer:
					gridStringBuilder.WriteString(darkPlayerStyle.Render("X"))
				case LightPlayer:
					gridStringBuilder.WriteString(lightPlayerStyle.Render("O"))
				default:
					if slices.Contains(availablePoints, point) {
						gridStringBuilder.WriteString(availablePointStyle.Render(" "))
					} else {
						gridStringBuilder.WriteString(" ")
					}
				}
			}

			if j < len(row)-1 {
				gridStringBuilder.WriteString(" ")
			}
		}

		if i < len(m.grid)-1 {
			gridStringBuilder.WriteString("\n")
		}
	}

	infoText := make([]string, 0, 10)
	infoText = append(infoText, lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true).
		Render(fmt.Sprintf("%s (%s)'s turn", m.currentPlayer.String(), m.currentPlayer.toSymbol())))

	var scoreStringBuilder strings.Builder
	if scores[LightPlayer] == scores[DarkPlayer] {
		scoreStringBuilder.WriteString("Draw")
	} else if scores[DarkPlayer] > scores[LightPlayer] {
		scoreStringBuilder.WriteString(fmt.Sprintf("%s winning!", DarkPlayer))
	} else if scores[LightPlayer] > scores[DarkPlayer] {
		scoreStringBuilder.WriteString(fmt.Sprintf("%s winning!", LightPlayer))
	}
	scoreStringBuilder.WriteString(" - ")
	scoreStringBuilder.WriteString(fmt.Sprintf("%s: %d; %s: %d", DarkPlayer.String(), scores[DarkPlayer], LightPlayer.String(),
		scores[LightPlayer]))
	infoText = append(infoText, scoreStringBuilder.String())

	switch m.view {
	case PointSelection:
		infoText = append(infoText, "", "Choose where to place your disk")
		if slices.Contains(availablePoints, m.selectedPoint) {
			infoText = append(infoText, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00cc00")).
				Render("Can place disk here"))
			infoText = append(infoText, "", lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Render("arrow keys: move • enter: place tile • q: exit"))
		} else {
			infoText = append(infoText, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cc0000")).
				Render("Cannot place disk here"))
			infoText = append(infoText, "", lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Render("arrow keys: move • q: exit"))
		}
	case PointConfirmation:
		if len(m.disksFlipped) == 0 {
			infoText = append(infoText, "", "No disks flipped this time")
		} else {
			infoText = append(infoText, "", fmt.Sprintf("%s flipped %d disks!", m.currentPlayer, len(m.disksFlipped)))
		}
		infoText = append(infoText, "", lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("any key: continue"))
	}

	gridString := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		MarginRight(6).
		Render(gridStringBuilder.String())

	infoTextString := lipgloss.JoinVertical(lipgloss.Left, infoText...)

	return lipgloss.NewStyle().
		Padding(2, 6).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, gridString, infoTextString))
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
