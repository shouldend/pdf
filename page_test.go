package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/llgcode/draw2d/draw2dimg"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const ignoreSize float64 = 3

func TestTextHorizontal(t *testing.T) {
	file, reader, e := Open(`../1207698967.PDF`)
	assert.Equal(t, nil, e)
	defer file.Close()
	numPage := reader.NumPage()
	var curSection string
	var (
		yMin float64 = 60
		yMax float64 = 790
	)

	for i := 0; i < numPage; i++ {
		page := reader.Page(i + 1)
		rows, _ := page.GetTextByRow()
		for r, row := range rows {
			if row.Position < yMin || row.Position > yMax {
				continue
			}
			if r == len(rows) && row.Content.Len() == 1 {
				_, err := strconv.Atoi(row.Content[0].S)
				if err == nil {
					// 如果是单纯的数字，那么直接跳过
					continue
				}
			}
			for j, text := range row.Content {
				// 识别出区域
				// 识别是否是页码
				if text.S == " " && j == row.Content.Len()-1 {
					fmt.Println(curSection)
					curSection = ""
				} else {
					curSection = curSection + text.S
				}
			}
		}
	}
}

func TestZLib(t *testing.T) {
	content, e := hex.DecodeString("48898C55CB6EDB3010BC0BD03FEC313998DE25970F0186018A9480160DD0A2EA29E82187B4A706682FFDFDF2A1D869E288860E92B0AB99D999A5BD7CECBB69E9BB2F7D07D35D00D87F86C3617F173E4460733CC21803FCEEBB31F5EC67052461F9D17704982E028D2CD800130AA960F9D577A9FE07502896CEC0DFBEBB3F20053E7E87658BC9AE4C99E6849D40FFC7BEBF81DB33D05B14B7FFF4F0F4136E1E9F76DFBEDE5E80241C8494A09C118E5648B90939342D20D2AF302F5AA01A16686C33C94461B699C869AF9C54C7DD709091267436A20CAEC54E6DF6446B1A73D280337AB4999D290E344E437E4636236A575491644F517B728CF95DE13C5EA35036152A72C2DA6D85AC2CB3434AE274F188035E938E6AB35B27B46EEC81933E2B409EFC4B2F8A92699888826F29E1A6123652B8D6465EE1B86E33D972FCB77722AA81832B5963DACD166BFB47C76941C8A08C12FCEE7C3A1DDD1D254AADA8AE20FB93847C3F35AC59C8307229586FD72FD7858DB55017361786B942A534CBA6A7444B8399756DB0937E8EB722E488293FA8BAF33CBBE7CC6B21079F21732CE50B550E4E358E5ECBCE1AEC5C0AE9984D2FE5570A3A7328F2A541692F57A831B442B05787C028F0DDE8E5500D51930D6787CE062147B3CEEFEA7851973BDB3956B1B3B6B553C58B8E91591BDAFF3209FACD54F04F800100744A6E8C")
	reader, e := zlib.NewReader(bytes.NewReader(content))
	if e != nil {
		t.Fatal(e)
	}
	cb, _ := ioutil.ReadAll(reader)
	t.Log("\n" + string(cb) + "\n")
}

type RectSlice []Rect

func (r RectSlice) Len() int {
	return len(r)
}

func (r RectSlice) Less(i, j int) bool {
	if isEqual(r[i].Min.Y, r[j].Min.Y) {
		if isEqual(r[i].Max.Y, r[j].Max.Y) {
			if isEqual(r[i].Min.X, r[j].Min.X) {
				return r[i].Max.X < r[j].Max.X
			}
			return r[i].Min.X < r[j].Min.X
		}
		return r[i].Max.Y < r[j].Max.Y
	}
	return r[i].Min.Y < r[j].Min.Y
}

func (r RectSlice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func newRect(rect Rect) Rect {
	return Rect{Point{rect.Min.X, 1000 - rect.Max.Y}, Point{rect.Max.X, 1000 - rect.Min.Y}}
}

func TestImage(t *testing.T) {
	file, reader, _ := Open(`../34ac99b37e3059ffb564b2da204a55d6.pdf`)
	defer file.Close()
	page := reader.Page(10)
	value := page.Resources().Key("XObject")
	dicts := value.data.(dict)
	for k, v := range dicts {
		if strings.HasPrefix(string(k), "Image") {
			result := reader.resolve(page.V.ptr, v)
			reader := result.Reader()
			b, e := ioutil.ReadAll(reader)
			if e != nil {
				panic(e)
			}
			s, ok := result.data.(stream)
			if !ok {
				continue
			}
			h, w := int(s.hdr["Height"].(int64)), int(s.hdr["Width"].(int64))
			var softMask []byte = nil
			img := image.NewRGBA(image.Rect(0, 0, w, h))
			i := 0
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					alpha := uint8(255)
					if softMask != nil {
						alpha = softMask[y*w+x]
					}
					img.Set(x, y, color.NRGBA{R: b[i], G: b[i+1], B: b[i+2], A: alpha})
					i += 3
				}
			}
			file, _ := os.Create("10.png")
			if e = png.Encode(file, img); e != nil {
				fmt.Println(e)
			}
			file.Close()
			reader.Close()
		}
	}
}

func TestSplit(t *testing.T) {
	file, reader, _ := Open(`../34ac99b37e3059ffb564b2da204a55d6.pdf`)
	defer file.Close()
	for pageNo := 0; pageNo < reader.NumPage(); pageNo++ {
		page := reader.Page(pageNo + 1)
		box := page.MediaBox()
		var (
			bounds   [4]float64
			useBound = false
		)
		if !box.IsNull() {
			bounds[0] = box.Index(0).Float64()
			bounds[1] = 1000 - box.Index(3).Float64()
			bounds[2] = box.Index(2).Float64()
			bounds[3] = 1000 - box.Index(1).Float64()
			useBound = true
		}
		texts := page.Content().Text
		var slice RectSlice
		for _, rect := range page.Content().Rect {
			slice = append(slice, newRect(rect))
		}
		sort.Sort(slice)
		last := Rect{}
		var current = RectSlice{}
		for _, rect := range slice {
			if rect == last {
				continue
			}
			// 跳过media之外的
			if useBound && (rect.Min.Y < bounds[1] || rect.Max.Y > bounds[3]) {
				continue
			}
			// 跳过线宽的
			if isEqual(rect.Max.X, rect.Min.X) || isEqual(rect.Max.Y, rect.Min.Y) {
				continue
			}
			if rect.Min.Y-last.Max.Y >= ignoreSize {
				// new rect
				if current.Len() > 0 {
					dealRect(texts, current)
				}
				current = RectSlice{}
			}
			current = append(current, rect)
			last = rect
		}
		if current.Len() > 0 {
			drawRect([]RectSlice{current})
			dealRect(texts, current)
		}
	}
}

func dealRect(texts []Text, rs RectSlice) {
	rs = minRect(rs)
	// 收集所有的x，y
	var xl, yl []float64
	for _, rect := range rs {
		insertSlice(&xl, rect.Min.X)
		insertSlice(&xl, rect.Max.X)
		insertSlice(&yl, rect.Min.Y)
		insertSlice(&yl, rect.Max.Y)
	}
	if len(xl) < 3 {
		return
	}
	//rows := getRows(rs)
	//result := rows2Table(rows, xl, yl, texts)
	matrix := getMatrix(rs, xl, yl)
	result := matrix2Table(matrix, texts)
	fmt.Println(result)
}

func minRect(rects RectSlice) RectSlice {
	var result = RectSlice{}
	for idx, rect := range rects {
		var isIgnore = false
		for next := idx + 1; next < rects.Len(); next++ {
			nextRect := rects[next]
			if isLTE(rect.Min.Y, nextRect.Min.Y) && isLTE(rect.Min.X, nextRect.Min.X) &&
				isGTE(rect.Max.Y, nextRect.Max.Y) && isGTE(rect.Max.X, nextRect.Max.X) {
				isIgnore = true
				break
			}
		}
		if !isIgnore {
			result = append(result, rect)
		}
	}
	return result
}

func matrix2Table(matrix [][]*Rect, texts []Text) string {
	var (
		result             = `<table border="2" bordercolor="black" width="90%" cellspacing="0" cellpadding="5">` + "\n"
		trueRows, trueCols = len(matrix), len(matrix[0])
		nilRect            = &Rect{}
		processed          = map[*Rect]bool{nilRect: true}
	)
	for row := 0; row < trueRows; row++ {
		var curResult = "<tr>\n"
		for col := 0; col < trueCols; col++ {
			// 如果是空的，进行colspan和rowspan并填充
			if matrix[row][col] == nil {
				curResult += "<td"
				var (
					mc = col + 1
					mr = row + 1
				)
				for ; mc < trueCols; mc++ {
					if matrix[row][mc] != nil {
						break
					}
				}
				for ; mr < trueRows; mr++ {
					var b = false
					for mmc := col; mmc < mc; mmc++ {
						if matrix[mr][mmc] != nil {
							b = true
							break
						}
					}
					if b {
						break
					}
				}
				if mc-col > 1 {
					curResult += fmt.Sprintf(` colspan="%d"`, mc-col)
				}
				if mr-row > 1 {
					curResult += fmt.Sprintf(` rowspan="%d"`, mr-row)
				}
				for mmr := row; mmr < mr; mmr++ {
					for mmc := col; mmc < mc; mmc++ {
						matrix[mmr][mmc] = nilRect
					}
				}
				curResult += " />\n"
				continue
			}
			if _, exists := processed[matrix[row][col]]; exists {
				continue
			}
			processed[matrix[row][col]] = true
			// 获取范围
			var (
				mr = row + 1
				mc = col + 1
			)
			for ; mr < trueRows; mr++ {
				if matrix[mr][col] != matrix[row][col] {
					break
				}
			}
			for ; mc < trueCols; mc++ {
				if matrix[row][mc] != matrix[row][col] {
					break
				}
			}
			curResult += "<td"
			if mc-col > 1 {
				curResult += fmt.Sprintf(` colspan="%d"`, mc-col)
			}
			if mr-row > 1 {
				curResult += fmt.Sprintf(` rowspan="%d"`, mr-row)
			}
			curResult += ">"
			// 选择内容
			var lastY float64 = -1
			for _, text := range texts {
				if inRect(text, *matrix[row][col]) {
					if lastY > 0 && lastY != text.Y {
						curResult += "<br/>"
					}
					curResult += strings.TrimSpace(text.S)
					lastY = text.Y
				}
			}
			curResult += "</td>\n"
		}
		result += curResult
		result += "</tr>\n"
	}
	result += "</table>\n"
	return result
}

func rows2Table(rows []RectSlice, xl, yl []float64, texts []Text) string {
	var result = `<table border="2" bordercolor="black" width="90%" cellspacing="0" cellpadding="5">` + "\n"
	for _, rowRects := range rows {
		var curResult = "<tr>\n"
		for _, rowRect := range rowRects {
			// 判断colspan
			var (
				rowspan, colspan int
				xMinPos, xMaxPos int
				yMinPos, yMaxPos int
			)

			for i, y := range yl {
				if isEqual(y, rowRect.Min.Y) {
					yMinPos = i
				}
				if isEqual(y, rowRect.Max.Y) {
					yMaxPos = i
					break
				}
			}
			rowspan = yMaxPos - yMinPos
			for i, x := range xl {
				if isEqual(x, rowRect.Min.X) {
					xMinPos = i
				}
				if isEqual(x, rowRect.Max.X) {
					xMaxPos = i
					break
				}
			}
			colspan = xMaxPos - xMinPos
			curResult += `<td`
			if colspan > 1 {
				curResult += fmt.Sprintf(` colspan="%d"`, colspan)
			}
			if rowspan > 1 {
				curResult += fmt.Sprintf(` rowspan="%d"`, rowspan)
			}
			curResult += `>`
			// 选择内容
			var lastY float64 = -1
			for _, text := range texts {
				if inRect(text, rowRect) {
					if lastY != -1 && lastY != text.Y {
						curResult += "\n"
					}
					curResult += strings.TrimSpace(text.S)
					lastY = text.Y
				}
			}
			curResult += "</td>\n"
		}
		curResult += "</tr>\n"
		result += curResult
	}
	result += "</table>\n"
	return result
}

func getRows(slice RectSlice) (result []RectSlice) {
	var (
		last    = Rect{}
		current = RectSlice{}
	)
	for _, rect := range slice {
		if !isEqual(last.Min.Y, rect.Min.Y) {
			if current.Len() > 0 {
				result = append(result, current)
			}
			current = RectSlice{}
		}
		if isEqual(last.Min.X, rect.Min.X) {
			if current.Len() > 0 && last.Max.X > rect.Max.X {
				current = current[:current.Len()-1]
			} else {
				continue
			}
		}
		current = append(current, rect)
		last = rect
	}
	if current.Len() > 0 {
		result = append(result, current)
	}
	return
}

func getMatrix(slice RectSlice, xl, yl []float64) [][]*Rect {
	sx, sy := len(xl), len(yl)
	result := make([][]*Rect, sy-1, sy-1)
	for i := 0; i < sy-1; i++ {
		result[i] = make([]*Rect, sx-1, sx-1)
	}
	for _, rect := range slice {
		var (
			minRow = indexSlice(yl, rect.Min.Y)
			maxRow = indexSlice(yl, rect.Max.Y)
			minCol = indexSlice(xl, rect.Min.X)
			maxCol = indexSlice(xl, rect.Max.X)
		)
		r := rect
		for row := minRow; row < maxRow; row++ {
			for col := minCol; col < maxCol; col++ {
				result[row][col] = &r
			}
		}
	}
	return result
}

func indexSlice(slice []float64, value float64) int {
	for i, v := range slice {
		if isEqual(v, value) {
			return i
		}
	}
	return -1
}

func inSlice(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func insertSlice(slice *[]float64, value float64) {
	for i, v := range *slice {
		if isEqual(v, value) {
			return
		}
		if v > value {
			var ns []float64
			ns = append(ns, (*slice)[:i]...)
			ns = append(ns, value)
			ns = append(ns, (*slice)[i:]...)
			*slice = ns
			return
		}
	}
	*slice = append(*slice, value)
}

func inRect(text Text, rect Rect) bool {
	x := text.X
	y := 1000 - text.Y
	return x >= rect.Min.X && x < rect.Max.X-0.0001 && y >= rect.Min.Y && y < rect.Max.Y-0.0001
}

func isEqual(x, y float64) bool {
	return x-y <= ignoreSize && x-y >= -ignoreSize
}
func isLTE(x, y float64) bool {
	return x-y <= ignoreSize
}

func isGTE(x, y float64) bool {
	return x-y >= -ignoreSize
}

var idx = 0

func drawRect(rows []RectSlice) {
	img := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	gc := draw2dimg.NewGraphicContext(img)
	gc.SetStrokeColor(color.RGBA{0x44, 0x44, 0x44, 0xff})
	gc.SetFillColor(color.Transparent)
	gc.SetLineWidth(1)
	for _, row := range rows {
		for _, rect := range row {
			minX, minY, maxX, maxY := float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Max.X), float64(rect.Max.Y)
			gc.MoveTo(minX, minY)
			gc.LineTo(minX, maxY)
			gc.LineTo(maxX, maxY)
			gc.LineTo(maxX, minY)
			gc.LineTo(minX, minY)
			gc.FillStroke()
		}
	}
	gc.Close()
	draw2dimg.SaveToPngFile(fmt.Sprintf("rect_%d.png", idx), img)
	idx += 1
}
