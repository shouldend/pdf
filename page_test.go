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
	content, e := hex.DecodeString("789CAD555B8BDA40147E0FE43F9C475D7032E7CC2D0322684C4A4B17D636A50FB20F62D5FAB0BA2B81D27FDF99EC2EC66C9AA8486002B99CEF726ED1F8506CD78B6501C361342E8AC5F2F7EA17CCA37CFFFC18E57F9F57D1C362B3DD2D8AED7E371AC1649AC0240F832843B090AFC30081BB0B8184654A81918619C89FC280334B163893FE404B70D8343CFCF6290CE64311634AC60A89CA8A6C82A347C8BF8441EA905ED1A88E660C8BAB682E1C6909F972DE238EB65F0F50A72B3832142701FCCF43A19389A0744C3633229B5A4A1221851A532C2D4D13F38199A80736C4E4890FE07583D7D9830AAF17FF74C09951A091FCCDD8F2765885C1CF3BD8BDFD6A3853FEA52CE34A834C037FFF6E7D170633473EBD4F20FA4F2A27FBA2D83FB565F3688AF4A668E93379610E7B58F3FC183466BA1AF4D486EAA7CE3A04A958DC6A5C2916A2072FF33EF93C051E7D5DEC36D05BED063FBEF7DF74BD1C4B875C6D38BFAA592A910C7722A8C472E701BCD15EDA9F3AE2AC0114AF04D55233A1AE04A52B4195AB2C7125A6B81653C48CAE152AEB38B20987B86B8858F8BE6884699B2BB3B203DB82C7DA57635B784AF598922C9598A148B38C529988C9341BB58B537571AA095F72E3BBBC8D40A78DBA2B77FA032C6A973690E4C6E309AA51DCA047BD6426507F80A6C7CB13FB883DDB41D8D4396213471282A95B912C477E47C6E2F36829CB6271335A0D5BA883A63D8BA6B0C8E846345F37915F9C0D0BB2716CF2B3AA50CA725DDCA80C3B6779D73047D93CE38436EEACD0734D1BEBC6DE6C8F46E83DB8245E29E21F4176FB32")
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
	file, reader, err := Open(`../34ac99b37e3059ffb564b2da204a55d6.pdf`)
	if err != nil {
		t.Log(err)
		return
	}
	defer file.Close()
	for i := 0; i < reader.NumPage(); i++ {
		page := reader.Page(i + 1)
		for j, img := range page.Images() {
			file, err := os.Create(fmt.Sprintf("%d_%d.png", i, j))
			if err != nil {
				t.Fatal(err)
			}
			if err = img.WritePng(file); err != nil {
				t.Fatal(err)
			}
			file.Close()
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
