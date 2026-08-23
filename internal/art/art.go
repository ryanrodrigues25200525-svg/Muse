package art

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"
)

// Render converts an image file into ANSI truecolor half-block art.
// width = terminal columns, height = terminal rows (each row = 2 image pixels).
// Uses bilinear interpolation for smooth downscaling.
func Render(imagePath string, width, height int) (string, error) {
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("invalid render size: %dx%d", width, height)
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	if src.Bounds().Dx() == 0 || src.Bounds().Dy() == 0 {
		return "", fmt.Errorf("image has empty bounds")
	}

	// Each terminal row covers 2 pixel rows via the ▀ half-block trick
	px := resizeBilinear(src, width, height*2)

	var sb strings.Builder
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			top := toRGB(px.NRGBAAt(col, row*2))
			bot := toRGB(px.NRGBAAt(col, row*2+1))
			// ▀ upper-half block: fg = top pixel, bg = bottom pixel
			sb.WriteString(fmt.Sprintf(
				"\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				top[0], top[1], top[2],
				bot[0], bot[1], bot[2],
			))
		}
		sb.WriteString("\x1b[0m\n")
	}
	return strings.TrimSuffix(sb.String(), "\n"), nil
}

// resizeBilinear scales src to w×h using bilinear interpolation.
// Produces significantly smoother results than nearest-neighbour when downscaling.
func resizeBilinear(src image.Image, w, h int) *image.NRGBA {
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 || w <= 0 || h <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, w, h))
	}
	sw := float64(b.Dx())
	sh := float64(b.Dy())
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Map destination pixel centre to source space
			srcX := (float64(x)+0.5)*sw/float64(w) - 0.5
			srcY := (float64(y)+0.5)*sh/float64(h) - 0.5

			x0 := int(math.Floor(srcX))
			y0 := int(math.Floor(srcY))
			x1 := x0 + 1
			y1 := y0 + 1

			// Clamp to image bounds
			x0 = clamp(x0, 0, b.Dx()-1)
			y0 = clamp(y0, 0, b.Dy()-1)
			x1 = clamp(x1, 0, b.Dx()-1)
			y1 = clamp(y1, 0, b.Dy()-1)

			fx := srcX - math.Floor(srcX)
			fy := srcY - math.Floor(srcY)

			c00 := toFloat(src.At(b.Min.X+x0, b.Min.Y+y0))
			c10 := toFloat(src.At(b.Min.X+x1, b.Min.Y+y0))
			c01 := toFloat(src.At(b.Min.X+x0, b.Min.Y+y1))
			c11 := toFloat(src.At(b.Min.X+x1, b.Min.Y+y1))

			dst.SetNRGBA(x, y, color.NRGBA{
				R: lerp4(c00[0], c10[0], c01[0], c11[0], fx, fy),
				G: lerp4(c00[1], c10[1], c01[1], c11[1], fx, fy),
				B: lerp4(c00[2], c10[2], c01[2], c11[2], fx, fy),
				A: 255,
			})
		}
	}
	return dst
}

// lerp4 performs bilinear interpolation on four uint16 channel values.
func lerp4(c00, c10, c01, c11 float64, fx, fy float64) uint8 {
	v := c00*(1-fx)*(1-fy) + c10*fx*(1-fy) + c01*(1-fx)*fy + c11*fx*fy
	return uint8(clampF(v, 0, 255))
}

func toFloat(c color.Color) [4]float64 {
	r, g, b, a := c.RGBA()
	return [4]float64{
		float64(r >> 8),
		float64(g >> 8),
		float64(b >> 8),
		float64(a >> 8),
	}
}

func toRGB(c color.NRGBA) [3]uint8 {
	return [3]uint8{c.R, c.G, c.B}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
