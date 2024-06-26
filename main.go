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

var version = "dev"

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

type rules int

const (
	ReversiRules rules = iota
	OthelloRules
)

func (r rules) String() string {
	return [...]string{"Reversi", "Othello"}[r]
}

type grid [gridHeight][gridWidth]player

type view int

const (
	PointSelection view = iota
	PointSelectionComputer
	PointConfirmation
	TitleView
	QuitConfirmation
	GameOverView
	PassView
)

type playerMode int

const (
	OnePlayer playerMode = iota
	TwoPlayer
)

func (pm playerMode) String() string {
	return [...]string{"1-Player", "2-Player"}[pm]
}

type model struct {
	grid            grid
	selectedPoint   vector2d
	view            view
	currentPlayer   player
	disksFlipped    []vector2d
	windowSize      vector2d
	availablePoints []vector2d
	rules           rules
	playerMode      playerMode
}

func newGrid(r rules) *grid {
	var g grid

	for i := 0; i < gridHeight; i++ {
		for j := 0; j < gridWidth; j++ {
			g[i][j] = Blank
		}
	}

	if r == OthelloRules {
		g[3][3] = LightPlayer
		g[4][4] = LightPlayer
		g[3][4] = DarkPlayer
		g[4][3] = DarkPlayer
	}

	return &g
}

func createInitialModel(r rules, pm playerMode) model {
	initialPlayer := DarkPlayer
	g := *newGrid(r)

	return model{
		grid:            g,
		selectedPoint:   vector2d{3, 3},
		view:            TitleView,
		currentPlayer:   initialPlayer,
		disksFlipped:    make([]vector2d, 0),
		availablePoints: getAvailablePoints(g, initialPlayer, r),
		rules:           r,
		playerMode:      pm,
	}
}

func initialModel() model {
	return createInitialModel(OthelloRules, OnePlayer)
}

func (m model) Init() tea.Cmd {
	return nil
}

func isComputerTurn(m model) bool {
	if m.playerMode == OnePlayer && m.currentPlayer == LightPlayer {
		return true
	}

	return false
}

func flipSelectedPoint(m *model) {
	m.grid[m.selectedPoint.y][m.selectedPoint.x] = m.currentPlayer
}

func takeTurn(m *model) {
	if slices.Contains(m.availablePoints, m.selectedPoint) {
		flipSelectedPoint(m)

		pointsToFlip := getPointsToFlip(m.grid, m.selectedPoint, m.currentPlayer)
		flip(&m.grid, pointsToFlip, m.currentPlayer)
		m.disksFlipped = pointsToFlip

		m.view = PointConfirmation
	}
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
				takeTurn(&m)
			}
		case PointSelectionComputer:
			takeTurn(&m)
		case PointConfirmation:
			// Update current player *after* displaying PointConfirmation view
			m.currentPlayer = toggleCurrentPlayer(m.currentPlayer)

			// Update available points
			availablePointsByPlayer := make(map[player][]vector2d)
			availablePointsByPlayer[DarkPlayer] = getAvailablePoints(m.grid, DarkPlayer, m.rules)
			availablePointsByPlayer[LightPlayer] = getAvailablePoints(m.grid, LightPlayer, m.rules)
			m.availablePoints = availablePointsByPlayer[m.currentPlayer]

			// If no available moves for current player then it's game over (for Reversi) or skip turn (for Othello)
			// If no available moves for either player then it's game over
			// Otherwise continue game and switch to PointSelection view
			playersCanMove := make(map[player]bool)
			playersCanMove[DarkPlayer] = len(availablePointsByPlayer[DarkPlayer]) > 0
			playersCanMove[LightPlayer] = len(availablePointsByPlayer[LightPlayer]) > 0

			if !playersCanMove[DarkPlayer] && !playersCanMove[LightPlayer] {
				m.view = GameOverView
			} else if !playersCanMove[m.currentPlayer] {
				if m.rules == ReversiRules {
					m.view = GameOverView
				} else {
					m.view = PassView
				}
			} else {
				if isComputerTurn(m) {
					m.view = PointSelectionComputer
					m.selectedPoint = computeBestPoint(m)
					flipSelectedPoint(&m)
				} else {
					m.view = PointSelection
				}
			}
		case TitleView:
			switch msg.String() {
			case "r":
				m.rules = toggleRules(m.rules)
				return createInitialModel(m.rules, m.playerMode), nil
			case "p":
				m.playerMode = togglePlayerMode(m.playerMode)
			default:
				m.view = PointSelection
			}
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
				return createInitialModel(m.rules, m.playerMode), nil
			default:
				return m, tea.Quit
			}
		case PassView:
			m.currentPlayer = toggleCurrentPlayer(m.currentPlayer)
			m.view = PointSelection
			m.availablePoints = getAvailablePoints(m.grid, m.currentPlayer, m.rules)
		}
	case tea.WindowSizeMsg:
		m.windowSize = vector2d{
			x: msg.Width,
			y: msg.Height,
		}
	}

	return m, nil
}

func computeBestPoint(m model) vector2d {
	var bestPoint vector2d
	maxFlippedPointsCount := -1 // Initialising to -1 so `bestPoint` is always assigned even if `flippedPointsCount` is 0

	for _, p := range m.availablePoints {
		flippedPointsCount := len(getPointsToFlip(m.grid, p, m.currentPlayer))
		if flippedPointsCount > maxFlippedPointsCount {
			bestPoint = p
			maxFlippedPointsCount = flippedPointsCount
		}
	}

	return bestPoint
}

func toggleCurrentPlayer(currentPlayer player) player {
	if currentPlayer == DarkPlayer {
		return LightPlayer
	}

	return DarkPlayer
}

func toggleRules(r rules) rules {
	if r == ReversiRules {
		return OthelloRules
	}

	return ReversiRules
}

func togglePlayerMode(pm playerMode) playerMode {
	if pm == OnePlayer {
		return TwoPlayer
	}

	return OnePlayer
}

func getNonBlankPoints(g grid) []vector2d {
	nonBlankPoints := make([]vector2d, 0)
	for i, row := range g {
		for j, cell := range row {
			if cell != Blank {
				nonBlankPoints = append(nonBlankPoints, vector2d{j, i})
			}
		}
	}
	return nonBlankPoints
}

func getAvailablePoints(g grid, currentPlayer player, r rules) []vector2d {
	// Get all non-blank points in grid
	nonBlankPoints := getNonBlankPoints(g)

	// Using Reversi rules, the first 4 disks must be placed with the centre 2x2 square in the grid
	if r == ReversiRules && len(nonBlankPoints) < 4 {
		availablePoints := []vector2d{
			{3, 3},
			{4, 4},
			{3, 4},
			{4, 3},
		}

		// Keep only points that are blank and inside the grid
		filteredAvailablePoints := make([]vector2d, 0, len(availablePoints))
		for _, p := range availablePoints {
			if isPointInsideGrid(p) && g[p.y][p.x] == Blank {
				filteredAvailablePoints = append(filteredAvailablePoints, p)
			}
		}

		return filteredAvailablePoints
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

const accentColor1 = lipgloss.Color("63")
const accentColor2 = lipgloss.Color("105")

var darkPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#ffffff")).
	Background(lipgloss.Color("#000000"))

var lightPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(lipgloss.Color("#ffffff"))

var selectedDarkPlayerStyle = lipgloss.NewStyle().
	Underline(true).
	Bold(true).
	Foreground(accentColor2).
	Background(lipgloss.Color("#000000"))

var selectedLightPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(accentColor2)

var selectedBlankStyle = lipgloss.NewStyle().
	Background(accentColor2)

const highlightedColor = lipgloss.Color("#666666")

var highlightedDarkPlayerStyle = lipgloss.NewStyle().
	Foreground(highlightedColor).
	Background(lipgloss.Color("#000000"))

var highlightedLightPlayerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(highlightedColor)

var availablePointStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#404040"))

var secondaryTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241"))

var accent1TextStyle = lipgloss.NewStyle().
	Foreground(accentColor1).
	Bold(true)

var successTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#00cc00"))

var errorTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#cc0000"))

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
	maxTextWidth := m.windowSize.x - ((gridWidth * 2) - 1) - 14
	switch m.view {
	case TitleView:
		text = createTitleView(maxTextWidth, m.rules, m.playerMode)
	case QuitConfirmation:
		text = createQuitConfirmationView(maxTextWidth)
	case GameOverView:
		text = createGameOverView(m, scores, maxTextWidth)
	case PointSelection:
		text = createPointSelectionView(m, scores, maxTextWidth, false)
	case PointConfirmation:
		text = createPointConfirmationView(m, scores, maxTextWidth)
	case PassView:
		text = createPassView(m, maxTextWidth)
	case PointSelectionComputer:
		text = createPointSelectionView(m, scores, maxTextWidth, true)
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
			if (m.view == PointSelection || m.view == PointSelectionComputer) && point == m.selectedPoint {
				switch cell {
				case DarkPlayer:
					gridStringBuilder.WriteString(selectedDarkPlayerStyle.Render("X"))
				case LightPlayer:
					gridStringBuilder.WriteString(selectedLightPlayerStyle.Render("O"))
				default:
					gridStringBuilder.WriteString(selectedBlankStyle.Render(" "))
				}
			} else if (m.view == PointConfirmation && m.grid[point.y][point.x] != Blank && !slices.Contains(m.disksFlipped, point)) ||
				(m.view == PointSelectionComputer && m.grid[point.y][point.x] != Blank && point != m.selectedPoint) {
				switch cell {
				case DarkPlayer:
					gridStringBuilder.WriteString(highlightedDarkPlayerStyle.Render("X"))
				case LightPlayer:
					gridStringBuilder.WriteString(highlightedLightPlayerStyle.Render("O"))
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
		BorderForeground(accentColor1).
		MarginRight(6).
		Render(gridStringBuilder.String())
}

func createTitleView(maxWidth int, r rules, pm playerMode) string {
	title := fmt.Sprintf(` ____                         _ 
|  _ \ _____   _____ _ __ ___(_)
| |_) / _ \ \ / / _ \ '__/ __| |
|  _ <  __/\ V /  __/ |  \__ \ |
|_| \_\___| \_/ \___|_|  |___/_|  %s`,
		secondaryTextStyle.Render(version))

	textStrings := []string{
		"",
		createRadioButton([]playerMode{OnePlayer, TwoPlayer}, pm, "Player mode", "P"),
		createRadioButton([]rules{OthelloRules, ReversiRules}, r, "Rules", "R"),
		"",
		"Press any other key to start...",
		"",
		secondaryTextStyle.Render("p: toggle player mode • r: toggle rules • any other key: continue"),
	}
	text := lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))

	return lipgloss.JoinVertical(lipgloss.Left, title, text)
}

func createQuitConfirmationView(maxWidth int) string {
	textStrings := []string{
		"Are you sure you want to quit?",
		"",
		"Any game progress will be lost.",
		"",
		secondaryTextStyle.Render("enter: quit • any other key: cancel"),
	}

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

func createGameOverView(m model, scores map[player]int, maxWidth int) string {
	var resultString string
	if scores[LightPlayer] == scores[DarkPlayer] {
		resultString = "Tie!"
	} else if scores[DarkPlayer] > scores[LightPlayer] {
		resultString = fmt.Sprintf("%s won!", DarkPlayer)
	} else if scores[LightPlayer] > scores[DarkPlayer] {
		resultString = fmt.Sprintf("%s won!", LightPlayer)
	}

	scoreString := fmt.Sprintf("%s: %d; %s: %d", DarkPlayer.String(), scores[DarkPlayer], LightPlayer.String(),
		scores[LightPlayer])

	var infoString string
	if m.rules == ReversiRules {
		infoString = fmt.Sprintf("No available moves for %s.", m.currentPlayer)
	} else {
		infoString = "No available moves for either player."
	}

	textStrings := []string{
		accent1TextStyle.Render("Game over!"),
		"",
		infoString,
		"",
		resultString,
		scoreString,
		"",
		secondaryTextStyle.Render("enter: play again • any other key: quit"),
	}

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

func createPointSelectionView(m model, scores map[player]int, maxWidth int, isComputerTurn bool) string {
	textStrings := make([]string, 0, 7)

	textStrings = append(textStrings, createTurnText(m.currentPlayer))
	textStrings = append(textStrings, createGameStatusText(scores))
	textStrings = append(textStrings, "")

	if isComputerTurn {
		textStrings = append(textStrings, "Computer places disk here")
		textStrings = append(textStrings, "", secondaryTextStyle.Render("any key: continue"))
	} else {
		textStrings = append(textStrings, "Choose where to place your disk")

		if slices.Contains(m.availablePoints, m.selectedPoint) {
			textStrings = append(textStrings, successTextStyle.Render("Can place disk here"))
			textStrings = append(textStrings, "", secondaryTextStyle.Render("arrow keys: move • enter: place tile • q: exit"))
		} else {
			textStrings = append(textStrings, errorTextStyle.Render("Cannot place disk here"))
			textStrings = append(textStrings, "", secondaryTextStyle.Render("arrow keys: move • q: exit"))
		}
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
	textStrings = append(textStrings, "", secondaryTextStyle.Render("any key: continue"))

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

type radioButtonItem interface {
	comparable
	String() string
}

func createRadioButton[T radioButtonItem](options []T, selected T, label string, key string) string {
	var builder strings.Builder
	builder.WriteString(label)
	builder.WriteString(": ")
	for i, option := range options {
		if option == selected {
			builder.WriteString(lipgloss.NewStyle().
				Foreground(accentColor2).
				Render(option.String() + " [▪]"))
		} else {
			builder.WriteString(option.String() + " [ ]")
		}

		if i != len(options)-1 {
			builder.WriteString(";")
		}

		builder.WriteString(" ")
	}
	builder.WriteString(secondaryTextStyle.Render(fmt.Sprintf("(press %s)", strings.ToUpper(key))))

	return builder.String()
}

func createPassView(m model, maxWidth int) string {
	textStrings := make([]string, 0, 6)
	textStrings = []string{
		createTurnText(m.currentPlayer),
		fmt.Sprintf("No available moves for %s; skipping turn...", m.currentPlayer),
		"",
		secondaryTextStyle.Render("any key: continue"),
	}

	return lipgloss.NewStyle().
		Width(maxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, textStrings...))
}

func createTurnText(currentPlayer player) string {
	return accent1TextStyle.Render(fmt.Sprintf("%s (%s)'s turn", currentPlayer.String(), currentPlayer.toSymbol()))
}

func createGameStatusText(scores map[player]int) string {
	var scoreStringBuilder strings.Builder
	if scores[LightPlayer] == scores[DarkPlayer] {
		scoreStringBuilder.WriteString("Tie")
	} else if scores[DarkPlayer] > scores[LightPlayer] {
		scoreStringBuilder.WriteString(fmt.Sprintf("%s winning!", DarkPlayer))
	} else if scores[LightPlayer] > scores[DarkPlayer] {
		scoreStringBuilder.WriteString(fmt.Sprintf("%s winning!", LightPlayer))
	}
	scoreStringBuilder.WriteString("\n")
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
