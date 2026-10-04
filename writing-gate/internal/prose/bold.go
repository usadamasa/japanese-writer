package prose

// このファイルの判定と直し方の案は nanaism/yomiyasu の scripts/yomiyasu_lint.py
// (bold_problems とその補助関数) を Go へ移植したもの｡
//
// Copyright (c) 2026 nanaism
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// boldProblem は表示されない太字 1 件｡Found と Suggest は前後 4 文字を含む抜粋｡
// 直し方の案が見つからなければ Suggest は空になる｡
type boldProblem struct {
	Line    int
	Found   string
	Suggest string
	How     string
}

const (
	howBracket  = "かっこの内側だけを太字にする"
	howPunct    = "句読点を太字の外に出す"
	howTrim     = "太字の内側の空白を取る"
	howSpace    = "文字に接する側に半角スペースを入れる"
	howByHand   = "手で直す"
	noRune      = rune(-1)
	boldExcerpt = 30
)

var (
	boldASCIIPunct = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	boldBrackets   = map[rune]rune{
		'「': '」', '『': '』', '（': '）', '(': ')', '【': '】', '〔': '〕', '［': '］', '[': ']',
		'〈': '〉', '《': '》', '“': '”', '‘': '’', '＜': '＞',
	}
	// boldTrailPunct は太字の外へ出せる句読点。原典の「。、．，！？!?」に、
	// 半角で書かれた文書も点検できるよう「｡､」を足す。
	boldTrailPunct = "。、．，！？!?｡､"

	boldListRe      = pyRe(`^\s{0,3}(?:[*+-]|\d+[.)])\s+`)
	boldLeadWSRe    = pyRe(`^\s{0,3}`)
	boldFenceRe     = pyRe("^\\s{0,3}(`{3,}|~{3,})(.*)$")
	boldThematicRe  = pyRe(`^\s{0,3}(?:\*\s*(?:\*\s*){2,}|-\s*(?:-\s*){2,}|_\s*(?:_\s*){2,})\s*$`)
	boldTableSepRe  = pyRe(`^\s{0,3}\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)+\|?\s*$`)
	boldSetextRe    = pyRe(`^\s{0,3}(=+|-+)\s*$`)
	boldATXRe       = pyRe(`^\s{0,3}#{1,6}(\s+|$)`)
	boldBacktickRun = regexp.MustCompile("`+")
)

// pyRe は Python の \s (Unicode の空白) に合わせて \s を広げてから compile する｡
// Go の \s は ASCII の空白だけで、全角空白で字下げした行の判定が原典とずれるため｡
func pyRe(expr string) *regexp.Regexp {
	return regexp.MustCompile(strings.ReplaceAll(expr, `\s`, `[\s\v\x{85}\p{Z}]`))
}

func boldWS(r rune) bool { return r == noRune || unicode.IsSpace(r) }

// boldPunctGFM は GitHub の GFM が記号とみなす文字 (ASCII の記号と Unicode の P)｡
func boldPunctGFM(r rune) bool {
	return r != noRune && (strings.ContainsRune(boldASCIIPunct, r) || unicode.IsPunct(r))
}

// boldPunctNew は新しい CommonMark が記号とみなす文字 (Unicode の P と S)｡
func boldPunctNew(r rune) bool {
	return r != noRune && (unicode.IsPunct(r) || unicode.IsSymbol(r))
}

var boldPunctPreds = []func(rune) bool{boldPunctGFM, boldPunctNew}

// boldCanOpen は GFM と CommonMark の両方で ** が太字を開けるかを返す｡
func boldCanOpen(prev, next rune) bool {
	for _, p := range boldPunctPreds {
		if boldWS(next) || (p(next) && !boldWS(prev) && !p(prev)) {
			return false
		}
	}
	return true
}

// boldCanClose は GFM と CommonMark の両方で ** が太字を閉じられるかを返す｡
func boldCanClose(prev, next rune) bool {
	for _, p := range boldPunctPreds {
		if boldWS(prev) || (p(prev) && !boldWS(next) && !p(next)) {
			return false
		}
	}
	return true
}

func runeAt(text []rune, p int) rune {
	if p < 0 || p >= len(text) {
		return noRune
	}
	return text[p]
}

type span struct{ start, end int }

// boldCodeSpans はインラインコード (同じ数のバッククォートで閉じたもの) の範囲を返す｡
func boldCodeSpans(text []rune) []span {
	s := string(text)
	var runs []span
	for _, m := range boldBacktickRun.FindAllStringIndex(s, -1) {
		runs = append(runs, span{runeIndex(s, m[0]), runeIndex(s, m[1])})
	}
	var spans []span
	for k := 0; k < len(runs); k++ {
		r := runs[k]
		for m := k + 1; m < len(runs); m++ {
			if runs[m].end-runs[m].start == r.end-r.start {
				spans = append(spans, span{r.start, runs[m].end})
				k = m
				break
			}
		}
	}
	return spans
}

// runeIndex はバイト位置を rune 位置へ変換する｡
func runeIndex(s string, byteIdx int) int {
	return len([]rune(s[:byteIdx]))
}

// boldDelimiters は長さがちょうど 2 の * の並びのうち、コードの外にあり
// エスケープされていないものの位置を返す｡
func boldDelimiters(text []rune) []int {
	code := boldCodeSpans(text)
	var pos []int
	for i := 0; i < len(text); {
		if text[i] != '*' {
			i++
			continue
		}
		j := i
		for j < len(text) && text[j] == '*' {
			j++
		}
		if j-i == 2 && !inSpans(code, i) && countBackslashes(text, i)%2 == 0 {
			pos = append(pos, i)
		}
		i = j
	}
	return pos
}

func inSpans(spans []span, p int) bool {
	for _, s := range spans {
		if s.start <= p && p < s.end {
			return true
		}
	}
	return false
}

func countBackslashes(text []rune, p int) int {
	n := 0
	for k := p - 1; k >= 0 && text[k] == '\\'; k-- {
		n++
	}
	return n
}

type boldPair struct{ open, close int }

// boldPairs はブロック内の ** を開きと閉じの組にする｡
// 第 1 段は前後の空白だけで開閉を決めてスタックで組み、第 2 段は内側の空白のせいで
// 開閉の条件から外れた候補を隣どうしで組む｡
func boldPairs(text []rune) []boldPair {
	pos := boldDelimiters(text)
	used := map[int]bool{}
	var pairs []boldPair

	var stack []int
	for _, p := range pos {
		canOpen := !boldWS(runeAt(text, p+2))
		canClose := !boldWS(runeAt(text, p-1))
		switch {
		case canClose && len(stack) > 0:
			opener := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if opener+2 < p {
				pairs = append(pairs, boldPair{opener, p})
				used[opener], used[p] = true, true
			}
		case canOpen:
			stack = append(stack, p)
		}
	}

	var unpaired []int
	for _, p := range pos {
		if !used[p] {
			unpaired = append(unpaired, p)
		}
	}
	for idx := 0; idx < len(unpaired)-1; {
		p1, p2 := unpaired[idx], unpaired[idx+1]
		crossed := false
		for _, pr := range pairs {
			if (p1 < pr.open && pr.open < p2) || (p1 < pr.close && pr.close < p2) {
				crossed = true
				break
			}
		}
		if crossed {
			idx++
			continue
		}
		inner := text[p1+2 : p2]
		if strings.TrimSpace(string(inner)) != "" && (boldWS(inner[0]) || boldWS(inner[len(inner)-1])) {
			pairs = append(pairs, boldPair{p1, p2})
			idx += 2
			continue
		}
		idx++
	}

	sort.SliceStable(pairs, func(a, b int) bool { return pairs[a].open < pairs[b].open })
	return pairs
}

func boldPairOK(text []rune, pr boldPair) bool {
	return boldCanOpen(runeAt(text, pr.open-1), runeAt(text, pr.open+2)) &&
		boldCanClose(runeAt(text, pr.close-1), runeAt(text, pr.close+2))
}

// boldCloseOf は s の先頭のかっこに対応する閉じかっこの位置を返す｡無ければ -1｡
func boldCloseOf(s []rune) int {
	o, c, depth := s[0], boldBrackets[s[0]], 0
	for k, x := range s {
		switch x {
		case o:
			depth++
		case c:
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

type boldTry struct {
	middle []rune
	how    string
}

func concat(parts ...[]rune) []rune {
	var out []rune
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// boldFix は k 番目の太字 (pr) の直し方の案を返す｡直した結果を組み直して、
// k 番目の組が太字として表示される案だけを採る｡
func boldFix(text []rune, pr boldPair, k int) ([]rune, string) {
	i, j := pr.open, pr.close
	inner := text[i+2 : j]
	stars := []rune("**")
	var tries []boldTry
	if len(inner) >= 3 {
		if _, ok := boldBrackets[inner[0]]; ok && boldCloseOf(inner) == len(inner)-1 {
			tries = append(tries, boldTry{
				concat(inner[:1], stars, inner[1:len(inner)-1], stars, inner[len(inner)-1:]), howBracket,
			})
		}
	}
	if len(inner) >= 2 && strings.ContainsRune(boldTrailPunct, inner[len(inner)-1]) {
		tries = append(tries, boldTry{concat(stars, inner[:len(inner)-1], stars, inner[len(inner)-1:]), howPunct})
	}

	body := inner
	if boldWS(runeAt(text, i+2)) || boldWS(runeAt(text, j-1)) {
		body = []rune(strings.TrimSpace(string(inner)))
	}
	first, last := noRune, noRune
	if len(body) > 0 {
		first, last = body[0], body[len(body)-1]
	}
	left, right := []rune(""), []rune("")
	if !boldCanOpen(runeAt(text, i-1), first) {
		left = []rune(" ")
	}
	if !boldCanClose(last, runeAt(text, j+2)) {
		right = []rune(" ")
	}
	how := howSpace
	if string(body) != string(inner) && len(left) == 0 && len(right) == 0 {
		how = howTrim
	}
	tries = append(tries, boldTry{concat(left, stars, body, stars, right), how})

	for _, t := range tries {
		cand := concat(text[:i], t.middle, text[j+2:])
		pairs := boldPairs(cand)
		if k < len(pairs) && boldPairOK(cand, pairs[k]) {
			return t.middle, t.how
		}
	}
	return nil, howByHand
}

// boldShort は長い太字の両端だけを残す｡
func boldShort(s []rune) string {
	if len(s) <= boldExcerpt {
		return string(s)
	}
	return string(s[:12]) + "…" + string(s[len(s)-12:])
}

type boldLine struct {
	num  int
	text string
}

// boldContainers は行頭のリストマーカーと引用の深さを読み、残りの本文を返す｡
func boldContainers(line string) (isList bool, depth int, content string) {
	rem := line
	if loc := boldListRe.FindStringIndex(line); loc != nil {
		isList = true
		rem = line[loc[1]:]
	}
	p := 0
	for {
		p += len(boldLeadWSRe.FindString(rem[p:]))
		if p < len(rem) && rem[p] == '>' {
			depth++
			p++
			if p < len(rem) && rem[p] == ' ' {
				p++
			}
			continue
		}
		break
	}
	return isList, depth, rem[p:]
}

// boldBlocks は太字の組を探す単位 (段落・リスト項目・見出し・表の行) へ行を分ける｡
// 強調は段落をまたげないため、空行・フェンス・HTML の行・区切り線・見出し・表・
// リスト項目の始まり・引用の深さの変化で区切る｡
func boldBlocks(src string) [][]boldLine {
	lines := strings.Split(src, "\n")
	start := 0
	if len(lines) > 0 && strings.TrimRight(lines[0], "\r") == "---" {
		for n := 1; n < len(lines); n++ {
			if strings.TrimRight(lines[n], "\r") == "---" {
				start = n + 1
				break
			}
		}
	}

	var blocks [][]boldLine
	var cur []boldLine
	var fenceChar byte
	fenceLen := 0
	curDepth := 0
	inTable := false

	flush := func() {
		if len(cur) > 0 {
			blocks = append(blocks, cur)
			cur = nil
		}
		curDepth = 0
		inTable = false
	}

	for no := start; no < len(lines); no++ {
		line := strings.TrimRight(lines[no], "\r")
		ln := boldLine{no + 1, line}
		isList, depth, content := boldContainers(line)

		m := boldFenceRe.FindStringSubmatch(content)
		if fenceLen > 0 {
			if m != nil && m[1][0] == fenceChar && len(m[1]) >= fenceLen && strings.TrimSpace(m[2]) == "" {
				fenceLen = 0
			}
			continue
		}
		// バッククォートのフェンスは info string にバッククォートを含められない
		if m != nil && (m[1][0] != '`' || !strings.Contains(m[2], "`")) {
			flush()
			fenceChar, fenceLen = m[1][0], len(m[1])
			continue
		}

		if strings.TrimSpace(content) == "" {
			flush()
			continue
		}
		if strings.HasPrefix(strings.TrimLeftFunc(content, unicode.IsSpace), "<") {
			flush()
			continue
		}
		if boldThematicRe.MatchString(content) {
			flush()
			continue
		}

		if boldTableSepRe.MatchString(content) {
			if len(cur) > 0 {
				hdr := cur[len(cur)-1]
				cur = cur[:len(cur)-1]
				flush()
				blocks = append(blocks, []boldLine{hdr})
			} else {
				flush()
			}
			inTable = true
			continue
		}

		if len(cur) > 0 && boldSetextRe.MatchString(content) {
			flush()
			continue
		}

		if boldATXRe.MatchString(content) {
			flush()
			blocks = append(blocks, []boldLine{ln})
			continue
		}

		if strings.HasPrefix(content, "|") || (inTable && strings.Contains(content, "|")) {
			flush()
			blocks = append(blocks, []boldLine{ln})
			inTable = true
			continue
		}
		inTable = false

		if isList || boldListRe.MatchString(content) {
			flush()
			curDepth = depth
			cur = append(cur, ln)
			continue
		}

		// 引用の深さが変わったら別の段落｡ただし引用の後の深さ 0 の行は
		// 遅延継続 (lazy continuation) として同じ段落に含める｡
		if len(cur) > 0 && depth != curDepth && (curDepth == 0 || depth != 0) {
			flush()
			curDepth = depth
		}
		if len(cur) == 0 {
			curDepth = depth
		}
		cur = append(cur, ln)
	}
	flush()
	return blocks
}

// boldProblems は GitHub などで太字として表示されない ** の位置と直し方の案を返す｡
// CommonMark (記号に Unicode の S を含む) と GFM (P のみ) の両方で太字になる形だけを
// 表示されるとみなす｡コードブロック・インラインコード・HTML の行・フロントマターは見ない｡
func boldProblems(src string) []boldProblem {
	var out []boldProblem
	for _, block := range boldBlocks(src) {
		var text []rune
		offsets := make([]int, 0, len(block))
		for n, ln := range block {
			if n > 0 {
				text = append(text, '\n')
			}
			offsets = append(offsets, len(text))
			text = append(text, []rune(ln.text)...)
		}
		lineOf := func(p int) int {
			idx := sort.Search(len(offsets), func(n int) bool { return offsets[n] > p }) - 1
			return block[idx].num
		}

		for k, pr := range boldPairs(text) {
			if boldPairOK(text, pr) {
				continue
			}
			middle, how := boldFix(text, pr, k)
			pre := string(text[max(0, pr.open-4):pr.open])
			post := string(text[pr.close+2 : min(len(text), pr.close+6)])
			p := boldProblem{
				Line:  lineOf(pr.open),
				Found: pre + boldShort(text[pr.open:pr.close+2]) + post,
				How:   how,
			}
			if middle != nil {
				p.Suggest = pre + boldShort(middle) + post
			}
			out = append(out, p)
		}
	}
	return out
}
