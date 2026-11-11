module github.com/usadamasa/japanese-writer

// net/http と cgo のビルド時の脆弱性は go1.27.4 で修正された。toolchain 行は GOTOOLCHAIN=local で効かないので、
// go 行を下限にして利用者の手元のビルドも修正版に限る
go 1.27.4
