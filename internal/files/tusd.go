package files

import (
	"github.com/rs/zerolog/log"
	"github.com/tus/tusd/v2/pkg/filelocker"
	"github.com/tus/tusd/v2/pkg/filestore"
	"github.com/tus/tusd/v2/pkg/handler"
)

// FileInfo represents file metadata
type FileInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploadedAt"`
}

// TusdInfoMetadata represents metadata in tusd .info files
type TusdInfoMetadata struct {
	Filename string `json:"filename"`
}

// TusdInfo represents the structure of tusd .info files
type TusdInfo struct {
	Size     int64            `json:"Size"`
	Offset   int64            `json:"Offset"`
	MetaData TusdInfoMetadata `json:"MetaData"`
}

// NewTusdHandler creates and configures a new tusd handler
func NewTusdHandler(filesDir string) (*handler.Handler, error) {
	store := filestore.New(filesDir)
	locker := filelocker.New(filesDir)

	composer := handler.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	cors := handler.DefaultCorsConfig
	cors.AllowCredentials = true

	tusHandler, err := handler.NewHandler(handler.Config{
		BasePath:                "/api/uploads/",
		StoreComposer:           composer,
		RespectForwardedHeaders: true,
		Cors:                       &cors,
		EnableExperimentalProtocol: true,
	})
	if err != nil {
		return nil, err
	}

	log.Info().Msg("Tusd handler initialized")
	return tusHandler, nil
}
