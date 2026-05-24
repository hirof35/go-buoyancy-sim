package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// 画面サイズ
	screenWidth  = 800
	screenHeight = 600

	// 物理パラメータ
	Gravity       = 9.81
	WaterDensity  = 1000.
	FluidDrag     = 0.5
	SphereRadius  = 25.0 // 画面上のピクセルサイズ
	SphereDensity = 500. // 水の密度の半分（よく浮く）

	// 波のパラメータ
	WaveBaseline  = 400.0 // 基準となる水面のY座標
	WaveAmplitude = 40.0  // 波の高さ（振幅）
	WaveFrequency = 0.015 // 波の細かさ
	WaveSpeed     = 0.05  // 波の進む速さ

	// 物理計算のタイムステップ（1フレームあたり）
	dt = 0.1
)

type Game struct {
	// 物体のステート
	x, y   float64 // 位置
	vx, vy float64 // 速度
	mass   float64 // 質量
	volume float64 // 体積
	ticks  float64 // 時間経過用カウンター
}

func NewGame() *Game {
	g := &Game{
		x:     100.0, // 左側からスタート
		y:     100.0, // 空中から落下
		ticks: 0,
	}
	// 2Dの円として面積（仮想的な体積）と質量を計算
	g.volume = math.Pi * math.Pow(SphereRadius, 2)
	g.mass = g.volume * SphereDensity
	return g
}

// GetWaveHeight は特定のX座標、時間(ticks)における水面のY座標を返します
func GetWaveHeight(x, ticks float64) float64 {
	return WaveBaseline + WaveAmplitude*math.Sin(WaveFrequency*x-WaveSpeed*ticks)
}

// GetWaveSlope は水面の傾き（微分係数）を返します
func GetWaveSlope(x, ticks float64) float64 {
	return WaveAmplitude * WaveFrequency * math.Cos(WaveFrequency*x-WaveSpeed*ticks)
}

// CalculateSubmergedVolume は球体が水に沈んでいる部分の断面積（体積）を計算
func CalculateSubmergedVolume(sphereY, radius, waveY float64) float64 {
	bottom := sphereY + radius
	top := sphereY - radius

	if bottom <= waveY { // 空中
		return 0.0
	}
	if top >= waveY { // 完全に水没
		return math.Pi * math.Pow(radius, 2)
	}

	// 部分的に沈んでいる（円欠の面積公式）
	h := bottom - waveY
	r := radius
	// 面積 = r^2 * acos((r-h)/r) - (r-h)*sqrt(2*r*h - h^2)
	area := math.Pow(r, 2)*math.Acos((r-h)/r) - (r-h)*math.Sqrt(2*r*h-math.Pow(h, 2))
	return area
}

// Update は毎フレーム（1/60秒ごと）の物理計算を行います
func (g *Game) Update() error {
	g.ticks += 1.0

	// 1. 現在位置の波の状態
	waveY := GetWaveHeight(g.x, g.ticks)
	slope := GetWaveSlope(g.x, g.ticks)

	// 2. 重力（Y軸は下方向がプラス）
	fg := g.mass * Gravity

	// 3. 浮力の計算
	submergedVol := CalculateSubmergedVolume(g.y, SphereRadius, waveY)
	buoyancyMagnitude := WaterDensity * submergedVol * Gravity

	// 波の傾き（斜面）に応じて浮力を分散（法線方向へ）
	theta := math.Atan(slope)
	fbx := buoyancyMagnitude * math.Sin(theta) // 水平方向の波力
	fby := buoyancyMagnitude * math.Cos(theta) // 垂直方向の浮力

	// 4. 水の抵抗（水に浸かっている割合に応じてブレーキ）
	dragX, dragY := 0.0, 0.0
	if submergedVol > 0 {
		ratio := submergedVol / g.volume
		dragX = FluidDrag * g.vx * math.Abs(g.vx) * ratio
		dragY = FluidDrag * g.vy * math.Abs(g.vy) * ratio
	}

	// 5. 合計の力（浮力fbyは上向きなのでマイナス）
	totalFx := fbx - dragX
	totalFy := fg - fby - dragY

	// 6. 移動（運動方程式 F=ma）
	ax := totalFx / g.mass
	ay := totalFy / g.mass

	g.vx += ax * dt
	g.vy += ay * dt
	g.x += g.vx * dt
	g.y += g.vy * dt

	// 画面外（右端）に出たら左端に戻すリピート処理
	if g.x > screenWidth+SphereRadius {
		g.x = -SphereRadius
		g.vx = 2.0 // 少し初速をもたせる
	}

	// 画面の下に落ちすぎないように壁判定
	if g.y > screenHeight-SphereRadius {
		g.y = screenHeight - SphereRadius
		g.vy = 0
	}

	return nil
}

// Draw は毎フレームの画面描画を担当します
func (g *Game) Draw(screen *ebiten.Image) {
	// 背景を薄いグレーで塗りつぶし
	screen.Fill(color.RGBA{240, 240, 243, 255})

	// 1. 水（波）の描画
	// 画面の左端から右端まで、1ピクセルずつ波の頂点を計算して縦線（または多角形）で塗る
	for x := 0.0; x < screenWidth; x++ {
		wy := GetWaveHeight(x, g.ticks)
		// 水面から画面最下部までを青色で描画
		vector.StrokeLine(screen, float32(x), float32(wy), float32(x), float32(screenHeight), 1.0, color.RGBA{41, 128, 185, 255}, false)
		// 水面の白いハイライトライン
		vector.DrawFilledRect(screen, float32(x), float32(wy), 1, 2, color.RGBA{133, 193, 233, 255}, false)
	}

	// 2. 浮遊する球体の描画
	// オレンジ色で物体を描画
	vector.DrawFilledCircle(screen, float32(g.x), float32(g.y), float32(SphereRadius), color.RGBA{230, 126, 34, 255}, true)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Buoyancy & Wave Simulation (Go)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}