package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"github.com/bmizerany/assert"
	"io/ioutil"
	"strconv"
	"testing"
)

func TestTextHorizontal(t *testing.T) {
	file, reader, e := Open(`/Users/donge/Desktop/1207698967.PDF`)
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
