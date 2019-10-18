package main

import (
	"github.com/signintech/gopdf"
	"math"
)

type Card struct {
	ImagePath string
	Name      string
}

type Page struct {
	Category string
	Cards    []Card
}

type Cache struct {
	EventName string
	Pages     []Page
}

func (p Page) Draw(event, filename string) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: *gopdf.PageSizeA4,
	})
	err := pdf.AddTTFFont("genshin", "GenShinGothic-P-Bold.ttf")
	if err != nil {
		panic(err)
	}
	cardsPerRow := 5
	cardsPerPage := 5 * 8
	cellSize := 99.0
	imageSize := 73.0
	mv := (gopdf.PageSizeA4.H - float64(cellSize*8)) / 2
	mh := (gopdf.PageSizeA4.W - float64(cellSize*5)) / 2
	var maxW float64
	for index, card := range p.Cards {
		if index%cardsPerPage == 0 {
			pdf.AddPage()
		}
		inPagePos := index % cardsPerPage
		row := inPagePos / cardsPerRow
		col := inPagePos % cardsPerRow
		offX := float64(col)*cellSize + mh
		offY := float64(row)*cellSize + mv
		pdf.SetLineWidth(0.1)
		pdf.RectFromUpperLeft(offX, offY, cellSize, cellSize)

		pdf.SetFont("genshin", "", 5)
		pdf.SetX(offX + 5)
		pdf.SetY(offY + 1)
		w, _ := pdf.MeasureTextWidth(event)
		pdf.SetX(offX + (cellSize-w)/2)

		pdf.Cell(nil, event)

		pdf.Image(card.ImagePath, offX+(cellSize-imageSize)/2, offY+6, &gopdf.Rect{
			W: imageSize,
			H: imageSize,
		})
		pdf.SetFont("genshin", "", 9)
		w, _ = pdf.MeasureTextWidth(card.Name)
		maxW = math.Max(maxW, w)
		pdf.SetX(offX + (cellSize-w)/2)
		pdf.SetY(offY + imageSize + 2 + 5)
		pdf.Cell(nil, card.Name)

		pdf.SetFont("genshin", "", 6)
		pdf.SetX(offX + 5)
		pdf.SetY(offY + cellSize - 7)
		pdf.Cell(nil, "Sponsored By")

		pdf.Image("logo-future.png", offX+(cellSize-47), offY+(cellSize-8), &gopdf.Rect{
			W: 37.42,
			H: 6,
		})
	}
	pdf.WritePdf(filename)
}
