package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize/english"
	"golang.org/x/exp/slices"
	"os"
	"strings"
)

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
	TitleView
	QuitConfirmation
	GameOverView
)

type model struct {
	grid            grid
	selectedPoint   vector2d
	view            view
	currentPlayer   player
	disksFlipped    []vector2d
	windowSize      vector2d
	availablePoints []vector2d
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
	g := *newGrid()
	return model{
		grid:            g,
		selectedPoint:   vector2d{3, 3},
		view:            TitleView,
		currentPlayer:   DarkPlayer,
		disksFlipped:    make([]vector2d, 0),
		availablePoints: getAvailablePoints(g, DarkPlayer),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.view {
		case PointSelection:
			switch msg.String() {
			case "ctrl+c", "q":
				m.view = QuitConfirmation
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
				if slices.Contains(m.availablePoints, m.selectedPoint) {
					m.grid[m.selectedPoint.y][m.selectedPoint.x] = m.currentPlayer

					pointsToFlip := getPointsToFlip(m.grid, m.selectedPoint, m.currentPlayer)
					flip(&m.grid, pointsToFlip, m.currentPlayer)
					m.disksFlipped = pointsToFlip

					// If no available points at the end of the turn, it's game over
					// Otherwise continue game and switch to PointConfirmation view
					availablePoints := getAvailablePoints(m.grid, m.currentPlayer)
					if len(availablePoints) == 0 {
						m.view = GameOverView
					} else {
						m.view = PointConfirmation
					}
				}
			}
		case PointConfirmation:
			m.view = PointSelection

			// Update current player *after* displaying PointConfirmation view
			if m.currentPlayer == DarkPlayer {
				m.currentPlayer = LightPlayer
			} else if m.currentPlayer == LightPlayer {
				m.currentPlayer = DarkPlayer
			}

			m.availablePoints = getAvailablePoints(m.grid, m.currentPlayer)
		case TitleView:
			m.view = PointSelection
		case QuitConfirmation:
			switch msg.String() {
			case "enter":
				return m, tea.Quit
			default:
				m.view = PointSelection
			}
		case GameOverView:
			switch msg.String() {
			case "enter":
				return initialModel(), nil
			default:
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		m.windowSize = vector2d{
			x: msg.Width,
			y: msg.Height,
		}
	}

	return m, nil
}

func getAvailablePoints(g grid, currentPlayer player) []vector2d {
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
	neighbors := make(map[vector2d]bool)
	for _, nonBlankPoint := range nonBlankPoints {
		for i := -1; i <= 1; i++ {
			for j := -1; j <= 1; j++ {
				if i != 0 || j != 0 {
					neighbor := vector2d{nonBlankPoint.x + j, nonBlankPoint.y + i}
					neighbors[neighbor] = true
				}
			}
		}
	}

	// Keep only neighbours that are blank, inside the grid and will result in at least one flipped point
	filteredNeighbors := make(map[vector2d]bool)
	for neighbor := range neighbors {
		if isPointInsideGrid(neighbor) && g[neighbor.y][neighbor.x] == Blank &&
			len(getPointsToFlip(g, neighbor, currentPlayer)) > 0 {
			filteredNeighbors[neighbor] = true
		}
	}

	filteredNeighborsList := make([]vector2d, 0, len(filteredNeighbors))
	for neighbor := range filteredNeighbors {
		filteredNeighborsList = append(filteredNeighborsList, neighbor)
	}
	return filteredNeighborsList
}

func isPointInsideGrid(p vector2d) bool {
	return p.x >= 0 && p.x < gridWidth && p.y >= 0 && p.y < gridHeight
}

func getPointsToFlip(g grid, selectedPoint vector2d, currentPlayer player) []vector2d {
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

	disksFlipped := make([]vector2d, 0, 10)
	for _, d := range directions {
		currentPoint := selectedPoint
		isInsideGrid := isPointInsideGrid(currentPoint)
		isNotBlank := true
		isCurrentPlayer := false
		pointsToFlip := make([]vector2d, 0)
		for isInsideGrid && isNotBlank && !isCurrentPlayer {
			currentPoint = vector2d{x: currentPoint.x + d.x, y: currentPoint.y + d.y}

			isInsideGrid = isPointInsideGrid(currentPoint)
			if !isInsideGrid {
				break
			}

			isNotBlank = g[currentPoint.y][currentPoint.x] != Blank
			isCurrentPlayer = g[currentPoint.y][currentPoint.x] == currentPlayer

			if isInsideGrid && isNotBlank && !isCurrentPlayer {
				pointsToFlip = append(pointsToFlip, currentPoint)
			}
		}

		// If disk of current player's colour is reached, change all the intermediate disks to the current player's colour
		// If blank cell or edge of grid is reached, don't change any disks
		if isCurrentPlayer {
			disksFlipped = append(disksFlipped, pointsToFlip...)
		}
	}

	return disksFlipped
}

func flip(g *grid, points []vector2d, currentPlayer player) {
	for _, p := range points {
		// Flip disk
		g[p.y][p.x] = currentPlayer
	}
}

var darkPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#ffffff")).
	Background(lipgloss.Color("#000000"))

var lightPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(lipgloss.Color("#ffffff"))

var selectedDarkPlayerStyle = lipgloss.NewStyle().
	Underline(true).
	Bold(true).
	Foreground(lipgloss.Color("105")).
	Background(lipgloss.Color("#000000"))

var selectedLightPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(lipgloss.Color("105"))

var selectedBlankStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("105"))

const highlightedColor = lipgloss.Color("#444444")

var highlightedDarkPlayerStyle = lipgloss.NewStyle().
	Foreground(highlightedColor).
	Background(lipgloss.Color("#000000"))

var highlightedLightPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(highlightedColor)

var highlightedBlankStyle = lipgloss.NewStyle().
	Background(highlightedColor)

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
	scores := computeScores(m.grid)

	gridString := createGridView(m)

	var text string
	maxTextWidth := m.windowSize.x - gridWidth - 14
	switch m.view {
	case TitleView:
		text = createTitleView(maxTextWidth)
	case QuitConfirmation:
		text = createQuitConfirmationView(maxTextWidth)
	case GameOverView:
		text = createGameOverView(scores, maxTextWidth)
	case PointSelection:
		text = createPointSelectionView(m, scores, maxTextWidth)
	case PointConfirmation:
		text = createPointConfirmationView(m, scores, maxTextWidth)
	}

	return lipgloss.NewStyle().
		Padding(2, 6).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, gridString, text))
}

func createGridView(m model) string {
	var gridStringBuilder strings.Builder
	for i, row := range m.grid {
		for j, cell := range row {
			point := vector2d{j, i}
			if m.view == PointSelection && point == m.selectedPoint {
				switch cell {
				case DarkPlayer:
					gridStringBuilder.WriteString(selectedDarkPlayerStyle.Render("X"))
				case LightPlayer:
					gridStringBuilder.WriteString(selectedLightPlayerStyle.Render("O"))
				default:
					gridStringBuilder.WriteString(selectedBlankStyle.Render(" "))
				}
			} else if m.view == PointConfirmation && m.grid[point.y][point.x] != Blank && !slices.Contains(m.disksFlipped, point) {
				switch cell {
				case DarkPlayer:
					gridStringBuilder.WriteString(highlightedDarkPlayerStyle.Render("X"))
				case LightPlayer:
					gridStringBuilder.WriteString(highlightedLightPlayerStyle.Render("O"))
				default:
					gridStringBuilder.WriteString(highlightedBlankStyle.Render(" "))
				}
			} else {
				switch cell {
				case DarkPlayer:
					gridStringBuilder.WriteString(darkPlayerStyle.Render("X"))
				case LightPlayer:
					gridStringBuilder.WriteString(lightPlayerStyle.Render("O"))
				default:
					if slices.Contains(m.availablePoints, point) {
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

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		MarginRight(6).
		Render(gridStringBuilder.String())
}

func createTitleView(maxWidth int) string {
	const title = ` ____                         _ 
|  _ \ _____   _____ _ __ ___(_)
| |_) / _ \ \ / / _ \ '__/ __| |
|  _ <  __/\ V /  __/ |  \__ \ |
|_| \_\___| \_/ \___|_|  |___/_|`

	textStrings := []string{
		"",
		"Press any key to start...",
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("any key: continue"),
	}
	text := lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))

	return lipgloss.JoinVertical(lipgloss.Left, title, text)
}

func createQuitConfirmationView(maxWidth int) string {
	textStrings := []string{
		"Are you sure you want to quit? Any game progress will be lost.",
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("enter: quit • any other key: cancel"),
	}

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

func createGameOverView(scores map[player]int, maxWidth int) string {
	var resultString string
	if scores[LightPlayer] == scores[DarkPlayer] {
		resultString = "Draw!"
	} else if scores[DarkPlayer] > scores[LightPlayer] {
		resultString = fmt.Sprintf("%s won!", DarkPlayer)
	} else if scores[LightPlayer] > scores[DarkPlayer] {
		resultString = fmt.Sprintf("%s won!", LightPlayer)
	}

	scoreString := fmt.Sprintf("%s: %d; %s: %d", DarkPlayer.String(), scores[DarkPlayer], LightPlayer.String(),
		scores[LightPlayer])

	textStrings := []string{
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Render("Game over!"),
		"",
		resultString,
		scoreString,
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("enter: play again • any other key: quit"),
	}

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

func createPointSelectionView(m model, scores map[player]int, maxWidth int) string {
	textStrings := make([]string, 0, 7)

	textStrings = append(textStrings, createTurnText(m.currentPlayer))
	textStrings = append(textStrings, createGameStatusText(scores))
	textStrings = append(textStrings, "", "Choose where to place your disk")

	if slices.Contains(m.availablePoints, m.selectedPoint) {
		textStrings = append(textStrings, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00cc00")).
			Render("Can place disk here"))
		textStrings = append(textStrings, "", lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("arrow keys: move • enter: place tile • q: exit"))
	} else {
		textStrings = append(textStrings, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cc0000")).
			Render("Cannot place disk here"))
		textStrings = append(textStrings, "", lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("arrow keys: move • q: exit"))
	}

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

func createPointConfirmationView(m model, scores map[player]int, maxWidth int) string {
	textStrings := make([]string, 0, 6)

	textStrings = append(textStrings, createTurnText(m.currentPlayer))
	textStrings = append(textStrings, createGameStatusText(scores))

	if len(m.disksFlipped) == 0 {
		textStrings = append(textStrings, "", "No disks flipped this time")
	} else {
		textStrings = append(textStrings, "", fmt.Sprintf("%s flipped %s!", m.currentPlayer, english.Plural(len(m.disksFlipped), "disk", "")))
	}
	textStrings = append(textStrings, "", lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("any key: continue"))

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

func createTurnText(currentPlayer player) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true).
		Render(fmt.Sprintf("%s (%s)'s turn", currentPlayer.String(), currentPlayer.toSymbol()))
}

func createGameStatusText(scores map[player]int) string {
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

	return scoreStringBuilder.String()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
