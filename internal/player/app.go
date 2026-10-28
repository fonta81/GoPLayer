// Package player proporciona la interfaz de usuario basada en terminal (TUI) con Bubble Tea,
// la gestión de eventos de teclado, escaneo de biblioteca y control del reproductor.
package player

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
)

// Tipos de mensaje internos utilizados por la arquitectura Elm de Bubble Tea.
type (
	// trackLoadedMsg se emite cuando un archivo de audio se decodifica y carga correctamente en el motor.
	trackLoadedMsg struct {
		track    Track
		duration time.Duration
	}
	// playbackEndedMsg se emite al terminar la reproducción del stream; sessionID evita procesar eventos antiguos.
	playbackEndedMsg    struct{ sessionID int }
	playbackPositionMsg struct {
		sessionID int
		position  time.Duration
	}
	// tickMsg se emite periódicamente para refrescar el tiempo transcurrido y la barra de progreso en la UI.
	tickMsg time.Time
	// libraryScannedMsg se emite tras completar el escaneo recursivo de un directorio musical.
	libraryScannedMsg struct {
		tracks []Track
		dir    string
	}
	// noMusicFoundMsg se emite cuando un escaneo no encuentra canciones compatibles; activa el explorador de carpetas.
	noMusicFoundMsg struct{ dir string }
	// errorMsg encapsula un error en tiempo de ejecución para mostrarlo temporalmente al usuario.
	errorMsg struct{ err error }
)

// AppModel es el modelo central de Bubble Tea que gestiona el estado del reproductor,
// las interacciones de usuario, la interfaz TUI y la integración con el motor de audio.
type AppModel struct {
	playlist *Playlist    // Lista de reproducción con control secuencial y aleatorio
	Audio    *AudioEngine // Motor de reproducción de bajo nivel

	state       PlaybackState // Estado de reproducción (detenido, reproduciendo, pausado)
	elapsed     time.Duration // Tiempo reproducido de la pista actual
	totalTime   time.Duration // Duración total de la pista actual
	cursorIndex int           // Posición del cursor en la lista de canciones
	volumeLevel float64       // Nivel de ganancia en decibelios (dB)
	lastError   error         // Último error capturado para la pantalla de alerta

	width       int            // Ancho actual del terminal
	height      int            // Alto actual del terminal
	progressBar progress.Model // Componente visual de la barra de progreso
	showHelp    bool           // Indica si se debe mostrar el panel de atajos de teclado
	showQueue   bool           // Indica si se debe mostrar la cola de pistas

	// Explorador interactivo de directorios
	musicDir        string         // Directorio de música actualmente configurado
	isPickingFolder bool           // Indica si el explorador de carpetas está en primer plano
	browserPath     string         // Ruta del directorio explorado en tiempo real
	browserEntries  []browserEntry // Elementos (carpetas y pistas compatibles) en browserPath
	browserCursor   int            // Posición del cursor en el explorador de carpetas

	// Búsqueda y filtrado rápido difuso (fuzzy filter)
	isFiltering      bool            // Indica si el campo de búsqueda rápida está activo
	filterInput      textinput.Model // Componente de entrada de texto interactivo
	filterQuery      string          // Texto ingresado por el usuario para filtrar
	filteredIndices  []int           // Índices de playlist.tracks que coinciden con la búsqueda
	filterCursor     int             // Posición del cursor dentro de filteredIndices
	searchCandidates []searchCandidate
	searchRevision   uint64

	// Ecualizador
	eqActive bool // Indica si el panel de ajuste del ecualizador tiene el foco
	eqCursor int  // Banda del ecualizador seleccionada actualmente

	// Lista de sugerencias destacadas para la búsqueda rápida
	suggestionLimit   int   // Límite máximo de sugerencias simultáneas a mostrar
	suggestionIndices []int // Índices de las mejores sugerencias calculadas
	suggestionCursor  int   // Índice de la sugerencia seleccionada en la lista emergente
}

// browserEntry representa un elemento individual (directorio o archivo de audio) en el explorador.
type browserEntry struct {
	name  string // Nombre del archivo o subdirectorio
	isDir bool   // True si el elemento es un directorio; false si es un archivo de audio
}

// NewAppModel construye e inicializa una nueva instancia de AppModel con valores predeterminados.
func NewAppModel(initialDir string) AppModel {
	bar := progress.New(progress.WithDefaultGradient())
	bar.Width = progressWidth
	bar.ShowPercentage = false

	ti := textinput.New()
	ti.Placeholder = "Buscar título o artista..."
	ti.CharLimit = 200
	ti.Width = 40

	return AppModel{
		playlist:        NewPlaylist(),
		Audio:           NewAudioEngine(),
		state:           StateStopped,
		progressBar:     bar,
		volumeLevel:     0,
		showHelp:        true,
		showQueue:       true,
		musicDir:        initialDir,
		filterInput:     ti,
		isFiltering:     false,
		filterQuery:     "",
		filteredIndices: nil,
		filterCursor:    0,
		// Configuración de sugerencias de búsqueda
		suggestionLimit:   6,
		suggestionIndices: nil,
		suggestionCursor:  0,
	}
}
