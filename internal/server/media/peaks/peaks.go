// Package peaks computes audio waveform peaks by shelling out to ffmpeg and
// parses HTTP Range headers for audio streaming.
package peaks

import (
	"bytes"
	"encoding/binary"
	"math"
	"os/exec"
	"strconv"
)

// ffmpegRunner shells out to ffmpeg. It is overridable in tests so that unit
// tests do not depend on a real ffmpeg binary.
type ffmpegRunner func(args []string) ([]byte, error)

var ffmpegRun ffmpegRunner = defaultFfmpegRun

func defaultFfmpegRun(args []string) ([]byte, error) {
	return exec.Command("ffmpeg", args...).Output()
}

// ComputePeaks runs ffmpeg to decode the audio file to mono f32le PCM, buckets
// the samples into the requested number of samples, and returns the maximum
// absolute amplitude per bucket rounded to four decimals. If any peak exceeds
// 1.0 the result is normalized by the maximum peak.
//
// Edge cases:
//   - ffmpeg failure or empty PCM data returns an empty slice.
//   - If the requested sample count exceeds the PCM sample count, every bucket
//     receives the global maximum absolute amplitude.
func ComputePeaks(audioPath string, samples int) ([]float64, error) {
	if samples <= 0 {
		return []float64{}, nil
	}

	targetAR := samples * 10 / 60
	if targetAR < 1000 {
		targetAR = 1000
	}

	args := []string{
		"-i", audioPath,
		"-f", "f32le",
		"-acodec", "pcm_f32le",
		"-ac", "1",
		"-ar", strconv.Itoa(targetAR),
		"-loglevel", "error",
		"-",
	}

	stdout, err := ffmpegRun(args)
	if err != nil {
		return []float64{}, nil
	}
	if len(stdout) == 0 {
		return []float64{}, nil
	}

	// Ignore trailing bytes that do not form a complete float32.
	count := len(stdout) / 4
	if count == 0 {
		return []float64{}, nil
	}
	pcm := make([]float32, count)
	if err := binary.Read(bytes.NewReader(stdout[:count*4]), binary.LittleEndian, pcm); err != nil {
		return []float64{}, nil
	}

	return pcmToPeaks(pcm, samples), nil
}

func pcmToPeaks(pcm []float32, samples int) []float64 {
	n := len(pcm)
	if n == 0 {
		return []float64{}
	}

	chunkSize := n / samples
	if chunkSize == 0 {
		maxAbs := maxAbsFloat32(pcm)
		peaks := make([]float64, samples)
		for i := range peaks {
			peaks[i] = maxAbs
		}
		return normalizePeaks(roundPeaks(peaks))
	}

	peaks := make([]float64, samples)
	for i := 0; i < samples; i++ {
		start := i * chunkSize
		end := start + chunkSize
		peaks[i] = maxAbsFloat32(pcm[start:end])
	}
	return normalizePeaks(roundPeaks(peaks))
}

func maxAbsFloat32(vals []float32) float64 {
	var max float64
	for _, v := range vals {
		abs := math.Abs(float64(v))
		if abs > max {
			max = abs
		}
	}
	return max
}

func roundPeaks(peaks []float64) []float64 {
	out := make([]float64, len(peaks))
	for i, v := range peaks {
		out[i] = math.Round(v*10000) / 10000
	}
	return out
}

func normalizePeaks(peaks []float64) []float64 {
	var max float64
	for _, v := range peaks {
		if v > max {
			max = v
		}
	}
	if max <= 1.0 {
		return peaks
	}
	out := make([]float64, len(peaks))
	for i, v := range peaks {
		out[i] = math.Round((v/max)*10000) / 10000
	}
	return out
}
