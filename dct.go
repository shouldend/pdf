package pdf

import (
	"bytes"
	"errors"
	"fmt"
	gocolor "image/color"
	"image/jpeg"
	"io"
	"io/ioutil"
)

func newDCTDecoder(reader io.Reader, space string, bits int) *DCTReader {
	d := &DCTReader{reader: reader, bitsPerComponent: bits}
	switch space {
	default:
		d.colorComponents = 1
	case "DeviceGray":
		d.colorComponents = 1
	case "DeviceRGB":
		d.colorComponents = 3
	case "DeviceCMYK":
		d.colorComponents = 4
	}
	return d
}

type DCTReader struct {
	colorComponents  int
	bitsPerComponent int
	reader           io.Reader
	bytesReader      io.Reader
	err              error
}

func (d *DCTReader) Read(p []byte) (n int, err error) {
	if d.err != nil {
		err = d.err
		return
	}
	if d.bytesReader == nil {
		if d.reader == nil {
			err = fmt.Errorf("nil reader")
			return
		}
		var (
			in  []byte
			out []byte
		)
		in, err = ioutil.ReadAll(d.reader)
		if err != nil {
			return
		}
		out, err = d.decode(in)
		if err != nil {
			return
		}
		d.bytesReader = bytes.NewReader(out)
	}
	n, err = d.bytesReader.Read(p)
	return
}

func (d *DCTReader) decode(in []byte) ([]byte, error) {
	bufReader := bytes.NewReader(in)
	//img, _, err := goimage.Decode(bufReader)
	img, err := jpeg.Decode(bufReader)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()

	var decoded = make([]byte, bounds.Dx()*bounds.Dy()*d.colorComponents*d.bitsPerComponent/8)
	index := 0

	for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
		for i := bounds.Min.X; i < bounds.Max.X; i++ {
			color := img.At(i, j)

			// Gray scale.
			if d.colorComponents == 1 {
				if d.bitsPerComponent == 16 {
					// Gray - 16 bit.
					val, ok := color.(gocolor.Gray16)
					if !ok {
						return nil, errors.New("color type error")
					}
					decoded[index] = byte((val.Y >> 8) & 0xff)
					index++
					decoded[index] = byte(val.Y & 0xff)
					index++
				} else {
					// Gray - 8 bit.
					val, ok := color.(gocolor.Gray)
					if !ok {
						return nil, errors.New("color type error")
					}
					decoded[index] = byte(val.Y & 0xff)
					index++
				}
			} else if d.colorComponents == 3 {
				if d.bitsPerComponent == 16 {
					val, ok := color.(gocolor.RGBA64)
					if !ok {
						return nil, errors.New("color type error")
					}
					decoded[index] = byte((val.R >> 8) & 0xff)
					index++
					decoded[index] = byte(val.R & 0xff)
					index++
					decoded[index] = byte((val.G >> 8) & 0xff)
					index++
					decoded[index] = byte(val.G & 0xff)
					index++
					decoded[index] = byte((val.B >> 8) & 0xff)
					index++
					decoded[index] = byte(val.B & 0xff)
					index++
				} else {
					// RGB - 8 bit.
					val, isRGB := color.(gocolor.RGBA)
					if isRGB {
						decoded[index] = val.R & 0xff
						index++
						decoded[index] = val.G & 0xff
						index++
						decoded[index] = val.B & 0xff
						index++
					} else {
						// Hack around YCbCr from go jpeg package.
						val, ok := color.(gocolor.YCbCr)
						if !ok {
							return nil, errors.New("color type error")
						}
						r, g, b, _ := val.RGBA()
						// The fact that we cannot use the Y, Cb, Cr values directly,
						// indicates that either the jpeg package is converting the raw
						// data into YCbCr with some kind of mapping, or that the original
						// data is not in R,G,B...
						// TODO: This is not good as it means we end up with R, G, B... even
						// if the original colormap was different.  Unless calling the RGBA()
						// call exactly reverses the previous conversion to YCbCr (even if
						// real data is not rgb)... ?
						// TODO: Test more. Consider whether we need to implement our own jpeg filter.
						decoded[index] = byte(r >> 8) //byte(val.Y & 0xff)
						index++
						decoded[index] = byte(g >> 8) //val.Cb & 0xff)
						index++
						decoded[index] = byte(b >> 8) //val.Cr & 0xff)
						index++
					}
				}
			} else if d.colorComponents == 4 {
				// CMYK - 8 bit.
				val, ok := color.(gocolor.CMYK)
				if !ok {
					return nil, errors.New("color type error")
				}
				// TODO: Is the inversion not handled right in the JPEG package for APP14?
				// Should not need to invert here...
				decoded[index] = 255 - val.C&0xff
				index++
				decoded[index] = 255 - val.M&0xff
				index++
				decoded[index] = 255 - val.Y&0xff
				index++
				decoded[index] = 255 - val.K&0xff
				index++
			}
		}
	}

	return decoded, nil
}
