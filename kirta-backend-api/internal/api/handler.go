package api

import (
	"context"
	"errors"
	"io"
	"kirta-backend-api/internal/domain"
	"kirta-backend-api/internal/service/svcerrs"
	"kirta-backend-api/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type ScanHandler struct {
	log  *slog.Logger
	scan ScanSvc
}

func NewScanHandler(log *slog.Logger, scan ScanSvc) *ScanHandler {
	return &ScanHandler{
		log:  log,
		scan: scan,
	}
}

type ScanSvc interface {
	Scan(ctx context.Context, pathToZip string) (domain.ScanInfo, error)
	GetScansBaseList(ctx context.Context) ([]domain.ScanBaseInfo, error)
	GetScanByID(ctx context.Context, id int64) (domain.ScanInfo, error)
	ExplainFinding(ctx context.Context, scanID int64, findingID int) (domain.ScaFinding, error)
	GetGraphByLibrary(ctx context.Context, scanID int64, packageName, version string) (domain.Graph, error)
	GetScanFile(ctx context.Context, scanID int64, filePath string) (io.ReadCloser, int64, error)
}

// StartScanScan godoc
// @Summary      Start a scan
// @Description  Upload a ZIP archive of a Python project. The service unpacks it, runs Syft (SBOM) + Grype (SCA) + the Python call-map tracer, persists the result and returns the full scan report.
// @Tags         scans
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "ZIP archive of the Python project"
// @Success      200  {object}  domain.ScanInfo
// @Failure      400  "file field missing"
// @Failure      500  "internal error"
// @Router       /v1/scan [post]
func (s *ScanHandler) StartScanScan(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		s.log.Error("get file from FormFile failed", "err", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}
	s.log.Info("start scan request", "file", f.Filename)
	tempDir, err := utils.CreateSafetyTempDir()
	if err != nil {
		s.log.Error("create temp dir failed", "err", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)
	dstFileName := filepath.Join(tempDir, f.Filename)
	if err := c.SaveUploadedFile(f, dstFileName); err != nil {
		s.log.Error("save file failed", "err", err.Error(), "file", f.Filename, "dir", tempDir)
		c.Status(http.StatusInternalServerError)
		return
	}
	s.log.Info("save file success", "file", dstFileName)
	report, err := s.scan.Scan(c.Request.Context(), dstFileName)
	if err != nil {
		var langErr svcerrs.LangErr
		if errors.As(err, &langErr) {
			s.log.Error("scan rejected: no python code found", "file", dstFileName)
			c.Status(http.StatusUnauthorized)
			return
		}
		s.log.Error("scan failed", "err", err.Error(), "file", dstFileName)
		c.Status(http.StatusInternalServerError)
		return
	}
	s.log.Info("scan success", "file", dstFileName, "id", report.ID)
	c.JSON(http.StatusOK, report)
}

// GetScansBaseList godoc
// @Summary      List all scans
// @Description  Returns a lightweight list of all scans ordered by creation time descending (no JSONB payload — id, repository, language, status, sloc, sha256, timestamps only).
// @Tags         scans
// @Produce      json
// @Success      200  {array}   domain.ScanBaseInfo
// @Failure      500  "internal error"
// @Router       /v1/scans [get]
func (s *ScanHandler) GetScansBaseList(c *gin.Context) {
	s.log.Info("get scans list request")
	scans, err := s.scan.GetScansBaseList(c.Request.Context())
	if err != nil {
		s.log.Error("get scans list failed", "err", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	s.log.Info("get scans list success", "count", len(scans))
	c.JSON(http.StatusOK, scans)
}

// GetScanByID godoc
// @Summary      Get scan by ID
// @Description  Returns the full scan report including language breakdown, library list, and SCA findings.
// @Tags         scans
// @Produce      json
// @Param        id  path  int  true  "Scan ID"
// @Success      200  {object}  domain.ScanInfo
// @Failure      400  "invalid id"
// @Failure      404  "scan not found"
// @Failure      500  "internal error"
// @Router       /v1/scans/{id} [get]
func (s *ScanHandler) GetScanByID(c *gin.Context) {
	rawID := c.Param("id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		s.log.Error("invalid scan id", "id", rawID, "err", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}
	s.log.Info("get scan by id request", "id", id)
	scan, err := s.scan.GetScanByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.Error("scan not found", "id", id)
			c.Status(http.StatusNotFound)
			return
		}
		s.log.Error("get scan by id failed", "id", id, "err", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	s.log.Info("get scan by id success", "id", id)
	c.JSON(http.StatusOK, scan)
}

// ExplainScanFinding godoc
// @Summary      Generate explanation for one finding
// @Description  Runs on-demand exploitability enrichment for a single SCA finding, persists updated finding in scan JSON and returns the updated finding.
// @Tags         scans
// @Produce      json
// @Param        id          path  int  true  "Scan ID"
// @Param        finding_id  path  int  true  "Finding ID"
// @Success      200  {object}  domain.ScaFinding
// @Failure      400  "invalid id"
// @Failure      404  "scan or finding not found"
// @Failure      500  "internal error"
// @Router       /v1/scans/{id}/findings/{finding_id}/explanation [post]
func (s *ScanHandler) ExplainScanFinding(c *gin.Context) {
	rawID := c.Param("id")
	scanID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		s.log.Error("invalid scan id", "id", rawID, "err", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}
	rawFindingID := c.Param("finding_id")
	findingID, err := strconv.Atoi(rawFindingID)
	if err != nil {
		s.log.Error("invalid finding id", "finding_id", rawFindingID, "err", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}

	finding, err := s.scan.ExplainFinding(c.Request.Context(), scanID, findingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, svcerrs.ErrFindingNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		s.log.Error("explain finding failed", "scan_id", scanID, "finding_id", findingID, "err", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, finding)
}

// GetScanGraphByLibrary godoc
// @Summary      Get call map by library
// @Description  Returns a call_map graph for a specific vulnerable library in the selected scan.
// @Tags         scans
// @Produce      json
// @Param        id       path   int     true   "Scan ID"
// @Param        package  query  string  true   "Library package name"
// @Param        version  query  string  false  "Library version"
// @Success      200  {object}  domain.Graph
// @Failure      400  "invalid id or empty package"
// @Failure      500  "internal error"
// @Router       /v1/scans/{id}/graphs [get]
func (s *ScanHandler) GetScanGraphByLibrary(c *gin.Context) {
	rawID := c.Param("id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		s.log.Error("invalid scan id", "id", rawID, "err", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}
	packageName := strings.TrimSpace(c.Query("package"))
	if packageName == "" {
		s.log.Error("empty package query", "id", id)
		c.Status(http.StatusBadRequest)
		return
	}
	version := strings.TrimSpace(c.Query("version"))

	s.log.Info("get scan graph by library request", "id", id, "package", packageName, "version", version)
	graph, err := s.scan.GetGraphByLibrary(c.Request.Context(), id, packageName, version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.Info("scan graph not found, return empty graph", "id", id, "package", packageName, "version", version)
			c.JSON(http.StatusOK, domain.Graph{
				ScanID:  id,
				Package: packageName,
				Version: version,
				CallMap: []domain.CallMap{},
			})
			return
		}
		s.log.Error("get scan graph by library failed", "id", id, "package", packageName, "version", version, "err", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	s.log.Info("get scan graph by library success", "id", id, "package", packageName, "version", version)
	c.JSON(http.StatusOK, graph)
}

// GetScanFile godoc
// @Summary      Download a source file from a scan
// @Description  Streams the raw source file that was referenced in a call_map entry. The file is fetched from MinIO S3 using the key scans/{id}/{filepath}.
// @Tags         scans
// @Produce      application/octet-stream
// @Param        id        path  int     true  "Scan ID"
// @Param        filepath  path  string  true  "Relative file path inside the project (e.g. src/utils.py)"
// @Success      200  {file}  binary  "Raw file contents"
// @Failure      400  "invalid id or empty path"
// @Failure      404  "file not found in storage"
// @Failure      500  "internal error"
// @Router       /v1/scans/{id}/files/{filepath} [get]
func (s *ScanHandler) GetScanFile(c *gin.Context) {
	rawID := c.Param("id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		s.log.Error("invalid scan id", "id", rawID, "err", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}
	filePath := strings.TrimPrefix(c.Param("filepath"), "/")
	if filePath == "" {
		s.log.Error("empty file path", "id", id)
		c.Status(http.StatusBadRequest)
		return
	}
	s.log.Info("get scan file request", "id", id, "file", filePath)
	reader, size, err := s.scan.GetScanFile(c.Request.Context(), id, filePath)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			s.log.Error("scan file not found", "id", id, "file", filePath)
			c.Status(http.StatusNotFound)
			return
		}
		s.log.Error("get scan file failed", "id", id, "file", filePath, "err", err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	s.log.Info("get scan file success", "id", id, "file", filePath, "size", size)
	c.DataFromReader(http.StatusOK, size, "application/octet-stream", reader, nil)
}
