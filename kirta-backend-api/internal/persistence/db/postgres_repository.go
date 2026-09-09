package db

import (
	"context"
	"encoding/json"
	"fmt"
	"kirta-backend-api/internal/domain"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScanRepository struct {
	db *pgxpool.Pool
}

func NewScanRepository(db *pgxpool.Pool) *ScanRepository {
	return &ScanRepository{
		db: db,
	}
}

func (s *ScanRepository) CreateScan(ctx context.Context, scan *domain.ScanInfo) error {
	libsJSON, err := json.Marshal(scan.Libraries)
	if err != nil {
		return fmt.Errorf("marshal libs: %w", err)
	}
	langsJSON, err := json.Marshal(scan.Langs)
	if err != nil {
		return fmt.Errorf("marshal langs: %w", err)
	}
	scaJSON, err := json.Marshal(scan.ScaReport)
	if err != nil {
		return fmt.Errorf("marshal sca: %w", err)
	}

	sqlQuery := `INSERT INTO kirta.scans (project_name, sloc, sha256, manifest, libs, languages, sca, status, created_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	return s.db.QueryRow(ctx, sqlQuery,
		scan.Repository, scan.TotalSLOC, scan.SHA256, scan.Manifest,
		libsJSON, langsJSON, scaJSON, scan.Status,
		scan.CreatedAt, scan.FinishedAt,
	).Scan(&scan.ID)
}

func (s *ScanRepository) GetScansBaseList(ctx context.Context) ([]domain.ScanBaseInfo, error) {
	sqlQuery := `SELECT id, project_name, status, sloc, sha256, created_at, finished_at
		FROM kirta.scans ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.ScanBaseInfo, error) {
		var scan domain.ScanBaseInfo
		err := row.Scan(
			&scan.ID, &scan.Repository, &scan.Status,
			&scan.TotalSLOC, &scan.SHA256, &scan.CreatedAt, &scan.FinishedAt,
		)
		return scan, err
	})
}

func (s *ScanRepository) GetScanByID(ctx context.Context, id int64) (domain.ScanInfo, error) {
	sqlQuery := `SELECT id, project_name, status, sloc, sha256, manifest, libs, languages, sca, created_at, finished_at
		FROM kirta.scans WHERE id = $1`

	var scan domain.ScanInfo
	var libsJSON, langsJSON, scaJSON []byte
	err := s.db.QueryRow(ctx, sqlQuery, id).Scan(
		&scan.ID, &scan.Repository, &scan.Status,
		&scan.TotalSLOC, &scan.SHA256, &scan.Manifest,
		&libsJSON, &langsJSON, &scaJSON,
		&scan.CreatedAt, &scan.FinishedAt,
	)
	if err != nil {
		return domain.ScanInfo{}, err
	}
	if err := json.Unmarshal(libsJSON, &scan.Libraries); err != nil {
		return domain.ScanInfo{}, fmt.Errorf("unmarshal libs: %w", err)
	}
	if err := json.Unmarshal(langsJSON, &scan.Langs); err != nil {
		return domain.ScanInfo{}, fmt.Errorf("unmarshal langs: %w", err)
	}
	if err := json.Unmarshal(scaJSON, &scan.ScaReport); err != nil {
		return domain.ScanInfo{}, fmt.Errorf("unmarshal sca: %w", err)
	}
	return scan, nil
}

func (s *ScanRepository) CreateGraphs(ctx context.Context, scanID int64, graphs []domain.Graph) error {
	sqlQuery := `INSERT INTO kirta.graphs (scan_id, package, version, call_map)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (scan_id, package, version)
		DO UPDATE SET call_map = EXCLUDED.call_map`

	for _, graph := range graphs {
		packageName := strings.TrimSpace(graph.Package)
		if packageName == "" {
			continue
		}
		version := strings.TrimSpace(graph.Version)

		callMapJSON, err := json.Marshal(graph.CallMap)
		if err != nil {
			return fmt.Errorf("marshal call_map: %w", err)
		}
		if _, err := s.db.Exec(
			ctx,
			sqlQuery,
			scanID,
			packageName,
			version,
			callMapJSON,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *ScanRepository) GetGraphByLibrary(
	ctx context.Context,
	scanID int64,
	packageName string,
	version string,
) (domain.Graph, error) {
	packageName = strings.TrimSpace(packageName)
	version = strings.TrimSpace(version)

	sqlQuery := `SELECT id, scan_id, package, version, call_map
		FROM kirta.graphs
		WHERE scan_id = $1
		  AND LOWER(package) = LOWER($2)
		  AND ($3 = '' OR version = $3)
		ORDER BY id DESC
		LIMIT 1`

	var graph domain.Graph
	var callMapJSON []byte
	err := s.db.QueryRow(ctx, sqlQuery, scanID, packageName, version).Scan(
		&graph.ID,
		&graph.ScanID,
		&graph.Package,
		&graph.Version,
		&callMapJSON,
	)
	if err != nil {
		return domain.Graph{}, err
	}
	if err := json.Unmarshal(callMapJSON, &graph.CallMap); err != nil {
		return domain.Graph{}, fmt.Errorf("unmarshal call_map: %w", err)
	}
	return graph, nil
}

func (s *ScanRepository) UpdateScanScaReport(ctx context.Context, scanID int64, scaReport domain.ScaReport) error {
	scaJSON, err := json.Marshal(scaReport)
	if err != nil {
		return fmt.Errorf("marshal sca: %w", err)
	}

	sqlQuery := `UPDATE kirta.scans SET sca = $1 WHERE id = $2`
	if _, err := s.db.Exec(ctx, sqlQuery, scaJSON, scanID); err != nil {
		return err
	}
	return nil
}
