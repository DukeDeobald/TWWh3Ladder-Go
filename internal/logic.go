package internal

import "math"

func expectedScore(ratingA int, ratingB int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(ratingB-ratingA)/400.0))
}

const K = 32

func updateElo(ratingWinner int, ratingLoser int, K int) (int, int) {
	newRatingWinner := float64(ratingWinner) + float64(K)*(1.0-expectedScore(ratingWinner, ratingLoser))
	newRatingLoser := float64(ratingLoser) + float64(K)*(0.0-expectedScore(ratingLoser, ratingWinner))
	return int(newRatingWinner), int(newRatingLoser)
}
