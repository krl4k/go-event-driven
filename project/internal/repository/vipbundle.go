package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"tickets/internal/entities"

	trmsqlx "github.com/avito-tech/go-transaction-manager/drivers/sqlx/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrVipBundleNotFound = fmt.Errorf("vip bundle not found")
var ErrVipBundleSkipped = fmt.Errorf("vip bundle update skipped")

type VipBundle struct {
	db     *sqlx.DB
	getter *trmsqlx.CtxGetter
}

func NewVipBundle(
	db *sqlx.DB,
	getter *trmsqlx.CtxGetter,
) *VipBundle {
	return &VipBundle{
		db:     db,
		getter: getter,
	}
}

func (vb *VipBundle) Add(ctx context.Context, vipBundle entities.VipBundle) error {
	bundleJson, err := json.Marshal(vipBundle)
	if err != nil {
		return fmt.Errorf("marshal vip bundle: %w", err)
	}

	_, err = vb.getter.DefaultTrOrDB(ctx, vb.db).ExecContext(ctx, `
		INSERT INTO vip_bundles (vip_bundle_id, booking_id, payload)
		VALUES ($1, $2, $3)
	`, vipBundle.VipBundleID, vipBundle.BookingID, bundleJson)

	if err != nil {
		return fmt.Errorf("insert vip bundle: %w", err)
	}
	return nil
}

func (vb *VipBundle) Get(ctx context.Context, vipBundleID uuid.UUID) (entities.VipBundle, error) {
	var vipBundle entities.VipBundle
	var bundleJson string
	err := vb.getter.DefaultTrOrDB(ctx, vb.db).QueryRowxContext(ctx, `
		SELECT payload
		FROM vip_bundles
		WHERE vip_bundle_id = $1
	`, vipBundleID).Scan(&bundleJson)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return vipBundle, ErrVipBundleNotFound
		}
		return vipBundle, fmt.Errorf("select vip bundle: %w", err)
	}

	err = json.Unmarshal([]byte(bundleJson), &vipBundle)
	if err != nil {
		return vipBundle, fmt.Errorf("unmarshal vip bundle: %w", err)
	}

	return vipBundle, nil
}

func (vb *VipBundle) GetByBookingID(ctx context.Context, bookingID uuid.UUID) (entities.VipBundle, error) {
	var vipBundle entities.VipBundle
	var bundleJson string
	err := vb.getter.DefaultTrOrDB(ctx, vb.db).QueryRowxContext(ctx, `
		SELECT payload
		FROM vip_bundles
		WHERE booking_id = $1
	`, bookingID).Scan(&bundleJson)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return vipBundle, ErrVipBundleNotFound
		}
		return vipBundle, fmt.Errorf("select vip bundle by booking id: %w", err)
	}

	err = json.Unmarshal([]byte(bundleJson), &vipBundle)
	if err != nil {
		return vipBundle, fmt.Errorf("unmarshal vip bundle: %w", err)
	}

	return vipBundle, nil
}

func (vb *VipBundle) UpdateByID(
	ctx context.Context,
	id uuid.UUID,
	updateFn func(vipBundle entities.VipBundle) (entities.VipBundle, error),
) (entities.VipBundle, error) {
	vipBundle, err := vb.Get(ctx, id)
	if err != nil {
		// if errors.Is(err, ErrVipBundleNotFound) {
		// 	log.Printf("skipping update: vip bundle with id %s doesn't exist", id)
		// 	return entities.VipBundle{}, ErrVipBundleSkipped
		// }
		return entities.VipBundle{}, err
	}

	vipBundle, err = updateFn(vipBundle)
	if err != nil {
		return entities.VipBundle{}, fmt.Errorf("update vip bundle: %w", err)
	}
	bundleJson, err := json.Marshal(vipBundle)
	if err != nil {
		return entities.VipBundle{}, fmt.Errorf("marshal vip bundle: %w", err)
	}

	_, err = vb.getter.DefaultTrOrDB(ctx, vb.db).ExecContext(ctx, `
		UPDATE vip_bundles
		SET payload = $1
		WHERE vip_bundle_id = $2
	`, bundleJson, vipBundle.VipBundleID)
	if err != nil {
		return entities.VipBundle{}, fmt.Errorf("update vip bundle: %w", err)
	}

	return vipBundle, nil
}

func (vb *VipBundle) UpdateByBookingID(
	ctx context.Context,
	bookingID uuid.UUID,
	updateFn func(vipBundle entities.VipBundle) (entities.VipBundle, error),
) (entities.VipBundle, error) {
	vipBundle, err := vb.GetByBookingID(ctx, bookingID)
	if err != nil {
		// if errors.Is(err, ErrVipBundleNotFound) {
		// 	log.Printf("skipping update: vip bundle with booking id %s doesn't exist", bookingID)
		// 	return entities.VipBundle{}, ErrVipBundleSkipped
		// }
		return entities.VipBundle{}, err
	}

	vipBundle, err = updateFn(vipBundle)
	if err != nil {
		return entities.VipBundle{}, fmt.Errorf("update vip bundle: %w", err)
	}

	bundleJson, err := json.Marshal(vipBundle)
	if err != nil {
		return entities.VipBundle{}, fmt.Errorf("marshal vip bundle: %w", err)
	}

	_, err = vb.getter.DefaultTrOrDB(ctx, vb.db).ExecContext(ctx, `
		UPDATE vip_bundles
		SET payload = $1
		WHERE booking_id = $2
	`, bundleJson, bookingID)

	if err != nil {
		return entities.VipBundle{}, fmt.Errorf("update vip bundle: %w", err)
	}
	return vipBundle, err
}
