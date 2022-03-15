package allstardata

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

const (
	Over5000 = -1
)

var (
	db *sql.DB

	specMap = map[string]int{
		"Astrologian": 1,
		"Bard":        2,
		"BlackMage":   3,
		"DarkKnight":  4,
		"Dragoon":     5,
		"Machinist":   6,
		"Monk":        7,
		"Ninja":       8,
		"Paladin":     9,
		"Scholar":     10,
		"Summoner":    11,
		"Warrior":     12,
		"WhiteMage":   13,
		"RedMage":     14,
		"Samurai":     15,
		"Dancer":      16,
		"Gunbreaker":  17,
	}
)

func init() {
	var err error

	db, err = sql.Open("sqlite3", "file:analysis/allstar/allstardata/allstardata.db?cache=shared&_fk=1&mode=ro")
	if err != nil {
		panic(err)
	}
}

type EncounterRank struct {
	Rank         int     // 순위
	RankPercent  float32 // 순위 % = (전체 - 순위 + 1) / 전체
	AllstarPoint float32 // GetEncounterRank 에서만 할당됨
}

func GetEncounterRank(ctx context.Context, encounterID int, partitionID int, spec string, rdps float32) (r EncounterRank, err error) {
	//log.Println("EncounterID:", encounterID, "partitionID:", partitionID, "spec:", spec, "rdps:", rdps)
	specInt := specMap[spec]

	var totalUsers int
	var maxRdps float32
	var minRdps float32
	err = db.QueryRowContext(
		ctx,
		`SELECT
			total_user,
			max_rdps,
			min_rdps
		FROM
			encounter_info
		WHERE
			encounter_id = ? AND
			partition_id = ? AND
			spec = ?
		LIMIT
			1`,
		encounterID,
		partitionID,
		specInt,
	).Scan(
		&totalUsers,
		&maxRdps,
		&minRdps,
	)
	if err != nil {
		return
	}
	r.AllstarPoint = rdps / maxRdps * 120

	if rdps <= minRdps {
		r.Rank = Over5000
		return
	}

	err = db.QueryRowContext(
		ctx,
		`SELECT
			1 + COUNT(*)
		FROM
			encounter
		WHERE
			encounter_id = ? AND
			partition_id = ? AND
			spec = ? AND
			rdps > ?
		LIMIT 1`,
		encounterID,
		partitionID,
		specInt,
		rdps,
	).Scan(
		&r.Rank,
	)
	if err != nil {
		return
	}

	r.RankPercent = float32(totalUsers-r.Rank+1) / float32(totalUsers) * 100
	return
}

func GetAllstarRank(ctx context.Context, zoneID int, partitionID int, spec string, allstar float32) (r EncounterRank, err error) {
	//log.Println("zoneID:", zoneID, "partitionID:", partitionID, "spec:", spec, "allstar:", allstar)
	specInt := specMap[spec]

	var totalUsers int
	var minAllstar float32
	err = db.QueryRowContext(
		ctx,
		`SELECT
			total_user,
			min_allstar
		FROM
			allstar_info
		WHERE
			zone_id = ? AND
			partition_id = ? AND
			spec = ?
		LIMIT
			1`,
		zoneID,
		partitionID,
		specInt,
	).Scan(
		&totalUsers,
		&minAllstar,
	)
	if err != nil {
		return
	}

	if allstar <= minAllstar {
		r.Rank = Over5000
		return
	}

	err = db.QueryRowContext(
		ctx,
		`SELECT
			1 + COUNT(*)
		FROM
			allstar
		WHERE
			zone_id = ? AND
			partition_id = ? AND
			spec = ? AND
			allstar > ?
		LIMIT 1`,
		zoneID,
		partitionID,
		specInt,
		allstar,
	).Scan(
		&r.Rank,
	)
	if err != nil {
		return
	}

	r.RankPercent = float32(totalUsers-r.Rank+1) / float32(totalUsers) * 100
	return
}
