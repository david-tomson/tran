package tui

import (
	"fmt"
	"strings"

	"github.com/abdfnx/tran/tools"
	"github.com/muesli/reflow/indent"
	"github.com/abdfnx/tran/constants"
	"github.com/muesli/reflow/wordwrap"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/progress"
)

type uiState int

// ui state flows from the top down
const (
	showEstablishing uiState = iota
	showReceivingProgress
	showFinished
	showError
)

type receiverUIModel struct {
	state                   uiState
	receivedFiles           []string
	payloadSize             int64
	decompressedPayloadSize int64
	spinner                 spinner.Model
	progressBar             progress.Model
	errorMessage            string
}

func NewReceiverUI() *tea.Program {
	m := receiverUIModel{
		progressBar: constants.ProgressBar,
	}

	m.resetSpinner()
	var opts []tea.ProgramOption

	opts = append(opts, tea.WithAltScreen())

	return tea.NewProgram(m, opts...)
}

func (receiverUIModel) Init() tea.Cmd {
	return spinner.Tick
}

func (m receiverUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case FileInfoMsg:
			m.payloadSize = msg.Bytes
			if m.state != showReceivingProgress {
				m.state = showReceivingProgress
				m.resetSpinner()

				return m, spinner.Tick
			}

			return m, nil

		case ProgressMsg:
			m.state = showReceivingProgress
			cmd := m.progressBar.SetPercent(float64(msg.Progress))

			return m, cmd

		case FinishedMsg:
			m.state = showFinished
			m.receivedFiles = msg.Files
			m.decompressedPayloadSize = msg.PayloadSize
			cmd := m.progressBar.SetPercent(1.0)

			return m, cmd

		case ErrorMsg:
			m.state = showError
			m.errorMessage = msg.Message

			return m, nil

		case tea.KeyMsg:
			if tools.Contains(constants.QuitKeys, strings.ToLower(msg.String())) {
				return m, tea.Quit
			}

			return m, nil

		case tea.WindowSizeMsg:
			m.progressBar.Width = msg.Width - 2 * constants.PADDING - 4
			if m.progressBar.Width > constants.MAX_WIDTH {
				m.progressBar.Width = constants.MAX_WIDTH
			}

			return m, nil

		// FrameMsg is sent when the progress bar wants to animate itself
		case progress.FrameMsg:
			progressModel, cmd := m.progressBar.Update(msg)
			m.progressBar = progressModel.(progress.Model)

			return m, cmd

		default:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)

			return m, cmd
	}
}

func (m receiverUIModel) View() string {
	switch m.state {
		case showEstablishing:
			return "\n" +
				constants.PadText + constants.InfoStyle(fmt.Sprintf("%s Establishing connection with sender", m.spinner.View())) + "\n\n"

		case showReceivingProgress:
			payloadSize := constants.BoldText(tools.ByteCountSI(m.payloadSize))
			receivingText := fmt.Sprintf("%s Receiving files (total size %s)", m.spinner.View(), payloadSize)

			return "\n" +
				constants.PadText + constants.InfoStyle(receivingText) + "\n\n" +
				constants.PadText + m.progressBar.View() + "\n\n" +
				constants.PadText + constants.QuitCommandsHelpText + "\n\n"

		case showFinished:
			payloadSize := constants.BoldText(tools.ByteCountSI(m.payloadSize))
			indentedWrappedFiles := indent.String(fmt.Sprintf("Received: %s", wordwrap.String(constants.ItalicText(TopLevelFilesText(m.receivedFiles)), constants.MAX_WIDTH)), constants.PADDING)
			finishedText := fmt.Sprintf("Received %d files (%s decompressed)\n\n%s", len(m.receivedFiles), payloadSize, indentedWrappedFiles)

			return "\n" +
				constants.PadText + constants.InfoStyle(finishedText) + "\n\n" +
				constants.PadText + m.progressBar.View() + "\n\n" +
				constants.PadText + constants.QuitCommandsHelpText + "\n\n"

		case showError:
			return m.errorMessage

		default:
			return ""
	}
}

func (m *receiverUIModel) resetSpinner() {
	m.spinner = spinner.NewModel()
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(constants.PRIMARY_COLOR))

	if m.state == showEstablishing {
		m.spinner.Spinner = WaitingSpinner
	}

	if m.state == showReceivingProgress {
		m.spinner.Spinner = TransferSpinner
	}
}
