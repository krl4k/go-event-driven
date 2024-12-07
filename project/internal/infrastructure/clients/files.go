package clients

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"net/http"
)

type FilesClient struct {
	clients *clients.Clients
}

func NewFilesClient(clients *clients.Clients) FilesClient {
	return FilesClient{
		clients: clients,
	}
}

func (c FilesClient) Upload(ctx context.Context, fileID string, content []byte) error {
	resp, err := c.clients.Files.PutFilesFileIdContentWithTextBodyWithResponse(ctx, fileID, string(content))
	if err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}

	if resp.StatusCode() == http.StatusConflict {
		log.FromContext(ctx).Infof("file %s already exists", fileID)
		return nil
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", resp.StatusCode())
	}

	return nil
}
