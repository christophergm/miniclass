package handlers

import (
	"context"
	"time"

	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/program"
)

// ArtifactService is separate so existing ProgramService implementations and
// fakes need not acquire artifact support.
type ArtifactService interface {
	GenerateArtifact(context.Context, string, ids.XID, ids.XID, ids.XID, program.ArtifactKind) (program.ArtifactDocument, error)
}
type ArtifactHandler struct{ service ArtifactService }

func NewArtifactHandler(service ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{service: service}
}

type ArtifactDocumentResponse struct {
	Kind        program.ArtifactKind      `json:"kind" enum:"class_list,homeroom_dismissal"`
	SessionName string                    `json:"session_name"`
	GeneratedAt time.Time                 `json:"generated_at"`
	Sections    []ArtifactSectionResponse `json:"sections"`
	Warnings    []ArtifactWarningResponse `json:"warnings"`
}
type ArtifactSectionResponse struct {
	ID     string                  `json:"id"`
	Title  string                  `json:"title"`
	Blocks []ArtifactBlockResponse `json:"blocks"`
}
type ArtifactBlockResponse struct {
	Kind  string `json:"kind" enum:"paragraph,heading,bullet,numbered"`
	Label string `json:"label" doc:"Bold label; empty when no label is needed."`
	Text  string `json:"text"`
}
type ArtifactWarningResponse struct {
	Code         string   `json:"code"`
	Message      string   `json:"message"`
	StudentNames []string `json:"student_names"`
}
type GenerateArtifactInput struct {
	SessionPathInput
	Kind program.ArtifactKind `path:"kind" enum:"class_list,homeroom_dismissal"`
}
type GenerateArtifactOutput struct{ Body ArtifactDocumentResponse }

func (h *ArtifactHandler) Generate(ctx context.Context, input *GenerateArtifactInput) (*GenerateArtifactOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, sessionNotFound()
	}
	doc, err := h.service.GenerateArtifact(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.SessionID), input.Kind)
	if err != nil {
		return nil, sessionProblem(err)
	}
	body := ArtifactDocumentResponse{Kind: doc.Kind, SessionName: doc.SessionName, GeneratedAt: doc.GeneratedAt, Sections: []ArtifactSectionResponse{}, Warnings: []ArtifactWarningResponse{}}
	for _, section := range doc.Sections {
		value := ArtifactSectionResponse{ID: section.ID, Title: section.Title, Blocks: []ArtifactBlockResponse{}}
		for _, block := range section.Blocks {
			value.Blocks = append(value.Blocks, ArtifactBlockResponse{Kind: block.Kind, Label: block.Label, Text: block.Text})
		}
		body.Sections = append(body.Sections, value)
	}
	for _, warning := range doc.Warnings {
		body.Warnings = append(body.Warnings, ArtifactWarningResponse{Code: warning.Code, Message: warning.Message, StudentNames: append([]string{}, warning.StudentNames...)})
	}
	return &GenerateArtifactOutput{Body: body}, nil
}
