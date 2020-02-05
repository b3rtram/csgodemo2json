package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	dem "github.com/markus-wa/demoinfocs-golang"
	com "github.com/markus-wa/demoinfocs-golang/common"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

type Game struct {
	MapName    string
	File       string
	FirstFrame int
	TickRate   int
}

type Frame struct {
	ID       int
	Mapname  int
	Player   []Player
	Grenades []Grenades
	Bomb     Vector
	Carrier  int
	Second   float64
	T        int
	CT       int
}

//FrPlayer is
type Player struct {
	Name     int      `json:"n"`
	Team     int      `json:"t"`
	Pos      Vector   `json:"pos"`
	View     Vector2D `json:"v"`
	Kills    int      `json:"k"`
	Deaths   int      `json:"d"`
	Assists  int      `json:"a"`
	MVPs     int      `json:"m"`
	Hp       int      `json:"hp"`
	Armor    int      `json:"ar"`
	Money    int      `json:"mo"`
	Score    int      `json:"sc"`
	Weapons  []Weapon `json:"ws"`
	Weapon   int      `json:"w"`
	Flashed  int      `json:"fl"`
	Footstep int      `json:"fo"`
	Visible  int      `json:in"`
}

type Weapon struct {
	Name int `json:"n"`
}

type Grenades struct {
	typ    int
	pos    Vector
	second float64
}

type Index struct {
	idx map[string]int
}

//Vector bla
type Vector struct {
	X float64
	Y float64
	Z float64
}

//Vector 2D
type Vector2D struct {
	X float32
	Y float32
}

func main() {

	input := flag.String("inputFile", "", "")
	//output := flag.String("outputPath", "", "")

	flag.Parse()

	c := make(chan Frame)
	//	go parseDemFile(c, *input)
	parseDemFile(c, *input)
	//writeFile(c, *output)

	select {}

}

func writeFile(c chan Frame, output string) {

	log.Println("create File")
	f, err := os.Create(output)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return
	}

	for {
		t := <-c
		j, _ := json.Marshal(t)
		f.Write(j)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func getIndex(idx *Index, value string) int {
	if val, ok := idx.idx[value]; ok {
		return val
	}

	newIdx := len(idx.idx) + 1
	idx.idx[value] = newIdx
	return newIdx
}

func parseDemFile(c chan Frame, file string) {

	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	h, _ := p.ParseHeader()
	log.Println(h.MapName)

	playerCache := make(map[int]*Player)

	game := Game{MapName: h.MapName, File: file}
	var frame Frame
	frame = Frame{}
	matchStarted := false
	index := Index{}

	game.TickRate = int(p.Header().TickRate())

	count := int64(0)
	oldsec := float64(0.0)

	curGran := make([]Grenades, 0)
	tRound := 0
	ctRound := 0

	p.RegisterEventHandler(func(e events.TickDone) {

		state := p.GameState()

		if !state.IsMatchStarted() {
			return
		}

		matchStarted = true

		count++

		players := state.Participants().Playing()
		frame.Player = make([]Player, len(players))

		for i := 0; i < len(players); i++ {
			frame.Player[i].Name = getIndex(&index, players[i].Name)

			if players[i].Team == com.TeamCounterTerrorists {
				frame.Player[i].Team = getIndex(&index, "Counter Terrorists")
			} else if players[i].Team == com.TeamTerrorists {
				frame.Player[i].Team = getIndex(&index, "Terrorists")
			}

			frame.Player[i].View = Vector2D{X: players[i].ViewDirectionX, Y: players[i].ViewDirectionY}
			frame.Player[i].Pos = Vector{X: players[i].Position.X, Y: players[i].Position.Y, Z: players[i].Position.Z}
			frame.Player[i].Armor = players[i].Armor
			frame.Player[i].Assists = players[i].AdditionalPlayerInformation.Assists
			frame.Player[i].Deaths = players[i].AdditionalPlayerInformation.Deaths
			frame.Player[i].Kills = players[i].AdditionalPlayerInformation.Kills
			frame.Player[i].MVPs = players[i].AdditionalPlayerInformation.MVPs
			frame.Player[i].Score = players[i].AdditionalPlayerInformation.Score
			frame.Player[i].Money = players[i].Money
			frame.Player[i].Hp = players[i].Hp

			frame.Player[i].Flashed = int(players[i].FlashDuration)

			frame.Player[i].Weapons = make([]Weapon, 0)
			if players[i].ActiveWeapon() != nil {
				frame.Player[i].Weapon = getIndex(&index, strings.Replace(players[i].ActiveWeapon().Weapon.String(), " ", "_", -1))
			}

			for k := range players[i].RawWeapons {
				weapon := Weapon{}
				// weapon.EntityID = players[i].RawWeapons[k].EntityID
				weapon.Name = getIndex(&index, strings.Replace(players[i].RawWeapons[k].Weapon.String(), " ", "_", -1))
				frame.Player[i].Weapons = append(frame.Player[i].Weapons, weapon)
			}

		}

		if state.Bomb().Carrier != nil {
			frame.Carrier = index.idx[state.Bomb().Carrier.Name]
		}

		frame.Bomb = Vector{X: state.Bomb().LastOnGroundPosition.X, Y: state.Bomb().LastOnGroundPosition.Y, Z: state.Bomb().LastOnGroundPosition.Z}
		frame.Second = p.CurrentTime().Seconds()
		frame.CT = ctRound
		frame.T = tRound
		frame.Grenades = curGran

		if frame.Second-oldsec > 0.04 {
			oldsec = frame.Second
			c <- frame
			frame = Frame{}
		}
	})

	p.RegisterEventHandler(func(e events.Kill) {
		if !matchStarted {
			return
		}
		playerCache[getIndex(&index, e.Victim.Name)].Visible = 0

	})

	p.RegisterEventHandler(func(e events.SmokeStart) {
		if !matchStarted {
			return
		}

		grenade := Grenades{}
		grenade.typ = getIndex(&index, e.GrenadeType.String())
		grenade.pos = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.second = p.CurrentTime().Seconds()

		curGran = append(curGran, grenade)
		log.Printf("Event Smoke Start")
	})

	// p.RegisterEventHandler(func(e events.SmokeExpired) {
	// 	if !matchStarted {
	// 		return
	// 	}

	// })

	p.RegisterEventHandler(func(e events.FlashExplode) {
		if !matchStarted {
			return
		}

		grenade := Grenades{}
		grenade.typ = getIndex(&index, e.GrenadeType.String())
		grenade.pos = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.second = p.CurrentTime().Seconds()

		curGran = append(curGran, grenade)

		log.Printf("Event Flash Explode")
	})

	p.RegisterEventHandler(func(e events.HeExplode) {
		if !matchStarted {
			return
		}

		grenade := Grenades{}
		grenade.typ = getIndex(&index, e.GrenadeType.String())
		grenade.pos = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.second = p.CurrentTime().Seconds()

		curGran = append(curGran, grenade)

		log.Printf("Event HE Explode")
	})

	p.RegisterEventHandler(func(e events.FireGrenadeStart) {
		if !matchStarted {
			return
		}

		grenade := Grenades{}
		grenade.typ = getIndex(&index, e.GrenadeType.String())
		grenade.pos = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.second = p.CurrentTime().Seconds()

		log.Printf("Event Fire Grenade Start")
	})

	// p.RegisterEventHandler(func(e events.FireGrenadeExpired) {
	// 	if !matchStarted {
	// 		return
	// 	}
	// 	grenade := EventGrenade{}
	// 	grenade.Grenade = e.GrenadeType.String()
	// 	grenade.Position = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
	// 	grenade.EventType = "FireGrenadeExpired"
	// 	tick.EventGrenade = append(tick.EventGrenade, grenade)
	// 	tick.Changes = true

	// 	log.Printf("Event Fire Grenade")
	// })

	// p.RegisterEventHandler(func(e events.ItemEquip) {
	// 	if !matchStarted {
	// 		return
	// 	}
	// 	itemEquip := EventItemEquip{}
	// 	itemEquip.Player = e.Player.Name
	// 	itemEquip.Weapon = e.Weapon.Weapon.String()
	// 	tick.EventItemEquip = append(tick.EventItemEquip, itemEquip)
	// 	tick.Changes = true

	// 	log.Printf("Event Item Equip")
	// })

	p.RegisterEventHandler(func(e events.RoundEnd) {
		if !matchStarted {
			return
		}
		if e.Winner == com.TeamCounterTerrorists {
			ctRound++
		} else if e.Winner == com.TeamTerrorists {
			tRound++
		}
	})

	// p.RegisterEventHandler(func(e events.Footstep) {
	// 	for _, element := range tick.Player {
	// 		if element.Name == e.Player.Name {
	// 			element.Footstep = true
	// 			break
	// 		}
	// 	}
	// })

	err = p.ParseToEnd()
	if err != nil {
		fmt.Printf("%s \n", err.Error())
	}

}
