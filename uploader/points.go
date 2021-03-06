package uploader

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/lomik/carbon-clickhouse/helper/RowBinary"
)

type Points struct {
	*Base
}

func NewPoints(base *Base) *Points {
	u := &Points{Base: base}
	u.Base.handler = u.upload
	return u
}

func (u *Points) upload(ctx context.Context, logger *zap.Logger, filename string) error {
	if u.config.ZeroTimestamp {
		reader, err := RowBinary.NewReader(filename)
		reader.SetZeroVersion(u.config.ZeroTimestamp)
		if err != nil {
			return err
		}

		return u.insertRowBinary(fmt.Sprintf("%s (Path, Value, Time, Date, Timestamp)", u.config.TableName), reader)
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	if fi.Size() == 0 {
		logger.Info("file is empty")
		return nil
	}
	err = u.insertRowBinary(
		fmt.Sprintf("%s (Path, Value, Time, Date, Timestamp)", u.config.TableName),
		file,
	)

	if err != nil {
		if strings.Contains(err.Error(), "Code: 33, e.displayText() = DB::Exception: Cannot read all data") {
			logger.Warn("file corrupted, try to recover")

			reader, err := RowBinary.NewReader(filename)
			if err != nil {
				return err
			}

			// try slow read method with skip bad records
			err = u.insertRowBinary(
				fmt.Sprintf("%s (Path, Value, Time, Date, Timestamp)", u.config.TableName),
				reader,
			)
			if err != nil {
				return err
			}
		}
	}

	return err
}
