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

type Ticks struct {
	Ticks []Tick
}

//Tick bla
type Tick struct {
	Second         float64          `json:"sec"`
	Player         []Player         `json:"p"`
	Bomb           Bomb             `json:"b"`
	EventKill      []EventKill      `json:"ek"`
	EventGrenade   []EventGrenade   `json:"eg"`
	EventRoundEnd  []EventRoundEnd  `json:"er"`
	EventItemEquip []EventItemEquip `json:"ei"`
	Changes        bool             `json:"c"`
}

//Event interface
type Event interface {
}

//EventKill is
type EventKill struct {
	Killer   string `json:"k"`
	Victim   string `json:"v"`
	Headshot bool   `json:"h"`
}

//EventGrenade is
type EventGrenade struct {
	Position  Vector `json:"p"`
	Thrower   string `json:"t"`
	Grenade   string `json:"g"`
	EventType string `json:"et"`
}

//EventItemEquip is
type EventItemEquip struct {
	Player string `json:"p"`
	Weapon string `json:"w"`
}

//EventRoundEnd is
type EventRoundEnd struct {
	Winner string `json:"w"`
	Looser string `json:"l"`
}

//Bomb is
type Bomb struct {
	Ground  Vector `json:"g"`
	Carrier Player `json:"c"`
}

//Weapon is
type Weapon struct {
	EntityID int    `json:"ei"`
	Name     string `json:"n"`
}

//Player bla
type Player struct {
	ID       int      `json:"id"`
	Name     string   `json:"n"`
	Team     string   `json:"t"`
	Pos      Vector   `json:"pos"`
	View     Vector2D `json:"v"`
	Kills    int      `json:"k"`
	Deaths   int      `json:"d"`
	Assists  int      `json:"a"`
	MVPs     int      `json:"m"`
	Hp       int      `json:"hp"`
	Armor    int      `json:"ar"`
	Money    int      `json:"mo"`
	SteamID  int64    `json:"st"`
	Score    int      `json:"sc"`
	Weapons  []Weapon `json:"ws"`
	Weapon   string   `json:"w"`
	Flashed  float32  `json:"fl"`
	Footstep bool     `json:"fo"`
}

//Game is
type Game struct {
	SumTicks int64   `json:"s"`
	Demo     string  `json:"d"`
	Map      string  `json:"m"`
	TickRate float64 `json:"tr"`
	Ticks    []Tick  `json:"ts"`
}

var ids map[string]int

func main() {

	input := flag.String("inputFile", "", "")
	//output := flag.String("outputPath", "", "")

	flag.Parse()

	c := make(chan Tick)
	//	go parseDemFile(c, *input)
	parseDemFile(c, *input)
	//writeFile(c, *output)

	select {}

}

func writeFile(c chan Tick, output string) {

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

func parseDemFile(c chan Tick, file string) {

	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	h, _ := p.ParseHeader()
	log.Println(h.MapName)
	game := Game{}
	var tick Tick
	tick = Tick{}
	matchStarted := false

	game.Map = p.Header().MapName
	game.Demo = file
	game.TickRate = p.Header().TickRate()

	ids = make(map[string]int)

	count := int64(0)

	oldsec := float64(0.0)

	p.RegisterEventHandler(func(e events.TickDone) {

		state := p.GameState()

		if !state.IsMatchStarted() {
			return
		}

		matchStarted = true

		count++

		players := state.Participants().Playing()
		tick.Player = make([]Player, len(players))

		for i := 0; i < len(players); i++ {

			tick.Changes = false
			tick.Player[i].Name = players[i].Name
			if players[i].Team == com.TeamCounterTerrorists {
				tick.Player[i].Team = "ct"
			} else if players[i].Team == com.TeamTerrorists {
				tick.Player[i].Team = "t"
			}

			if _, ok := ids[tick.Player[i].Name]; !ok {
				maxID := 0
				for _, k := range ids {
					if maxID < k {
						maxID = k
					}
				}
				ids[tick.Player[i].Name] = maxID + 1
			}

			tick.Player[i].View = Vector2D{X: players[i].ViewDirectionX, Y: players[i].ViewDirectionY}
			tick.Player[i].Pos = Vector{X: players[i].Position.X, Y: players[i].Position.Y, Z: players[i].Position.Z}
			tick.Player[i].Armor = players[i].Armor
			tick.Player[i].Assists = players[i].AdditionalPlayerInformation.Assists
			tick.Player[i].Deaths = players[i].AdditionalPlayerInformation.Deaths
			tick.Player[i].Kills = players[i].AdditionalPlayerInformation.Kills
			tick.Player[i].MVPs = players[i].AdditionalPlayerInformation.MVPs
			tick.Player[i].Score = players[i].AdditionalPlayerInformation.Score
			tick.Player[i].Money = players[i].Money
			tick.Player[i].Hp = players[i].Hp
			tick.Player[i].SteamID = players[i].SteamID
			tick.Player[i].ID = ids[players[i].Name]
			tick.Player[i].Flashed = players[i].FlashDuration

			tick.Player[i].Weapons = make([]Weapon, 0)
			if players[i].ActiveWeapon() != nil {
				tick.Player[i].Weapon = strings.Replace(players[i].ActiveWeapon().Weapon.String(), " ", "_", -1)
			}
			for k := range players[i].RawWeapons {
				weapon := Weapon{}
				weapon.EntityID = players[i].RawWeapons[k].EntityID
				weapon.Name = strings.Replace(players[i].RawWeapons[k].Weapon.String(), " ", "_", -1)

				tick.Player[i].Weapons = append(tick.Player[i].Weapons, weapon)
			}

		}

		tick.Bomb = Bomb{}
		tick.Bomb.Carrier = Player{}
		if state.Bomb().Carrier != nil {
			tick.Bomb.Carrier.Name = state.Bomb().Carrier.Name
			tick.Bomb.Carrier.ID = ids[state.Bomb().Carrier.Name]
		}

		tick.Bomb.Ground = Vector{X: state.Bomb().LastOnGroundPosition.X, Y: state.Bomb().LastOnGroundPosition.Y, Z: state.Bomb().LastOnGroundPosition.Z}
		tick.Second = p.CurrentTime().Seconds()

		//fmt.Println(oldsec)
		if tick.Second-oldsec > 0.2 || tick.Changes == true {
			oldsec = tick.Second
			fmt.Printf("time elapsed %t\n", tick.Changes)
		} else {
			//c <- Tick{Second: p.CurrentTime().Seconds()}
		}

		tick = Tick{}
	})

	p.RegisterEventHandler(func(e events.Kill) {
		if !matchStarted {
			return
		}
		kill := EventKill{}
		kill.Killer = e.Killer.Name
		kill.Victim = e.Victim.Name
		kill.Headshot = e.IsHeadshot
		tick.EventKill = append(tick.EventKill, kill)
		tick.Changes = true
		log.Printf("Event Kill")
	})

	p.RegisterEventHandler(func(e events.SmokeStart) {
		if !matchStarted {
			return
		}
		grenade := EventGrenade{}
		grenade.Grenade = e.GrenadeType.String()
		grenade.Position = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.Thrower = e.Thrower.Name
		grenade.EventType = "SmokeStart"
		tick.EventGrenade = append(tick.EventGrenade, grenade)
		tick.Changes = true
		log.Printf("Event Smoke Start")
	})

	p.RegisterEventHandler(func(e events.SmokeExpired) {
		if !matchStarted {
			return
		}
		grenade := EventGrenade{}
		grenade.Grenade = e.GrenadeType.String()
		grenade.Position = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.Thrower = e.Thrower.Name
		grenade.EventType = "SmokeExpired"
		tick.EventGrenade = append(tick.EventGrenade, grenade)
		tick.Changes = true
		log.Printf("Event Smoke Expired")
	})

	p.RegisterEventHandler(func(e events.FlashExplode) {
		if !matchStarted {
			return
		}
		grenade := EventGrenade{}
		grenade.Grenade = e.GrenadeType.String()
		grenade.Position = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.Thrower = e.Thrower.Name
		grenade.EventType = "FlashExplode"
		tick.EventGrenade = append(tick.EventGrenade, grenade)
		tick.Changes = true

		log.Printf("Event Flash Explode")
	})

	p.RegisterEventHandler(func(e events.HeExplode) {
		if !matchStarted {
			return
		}
		grenade := EventGrenade{}
		grenade.Grenade = e.GrenadeType.String()
		grenade.Position = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.Thrower = e.Thrower.Name
		grenade.EventType = "HeExplode"
		tick.EventGrenade = append(tick.EventGrenade, grenade)
		tick.Changes = true

		log.Printf("Event HE Explode")
	})

	p.RegisterEventHandler(func(e events.FireGrenadeStart) {
		if !matchStarted {
			return
		}
		grenade := EventGrenade{}
		grenade.Grenade = e.GrenadeType.String()
		grenade.Position = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.EventType = "FireGrenadeStart"
		tick.EventGrenade = append(tick.EventGrenade, grenade)
		tick.Changes = true

		log.Printf("Event Fire Grenade Start")
	})

	p.RegisterEventHandler(func(e events.FireGrenadeExpired) {
		if !matchStarted {
			return
		}
		grenade := EventGrenade{}
		grenade.Grenade = e.GrenadeType.String()
		grenade.Position = Vector{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z}
		grenade.EventType = "FireGrenadeExpired"
		tick.EventGrenade = append(tick.EventGrenade, grenade)
		tick.Changes = true

		log.Printf("Event Fire Grenade")
	})

	p.RegisterEventHandler(func(e events.ItemEquip) {
		if !matchStarted {
			return
		}
		itemEquip := EventItemEquip{}
		itemEquip.Player = e.Player.Name
		itemEquip.Weapon = e.Weapon.Weapon.String()
		tick.EventItemEquip = append(tick.EventItemEquip, itemEquip)
		tick.Changes = true

		log.Printf("Event Item Equip")
	})

	p.RegisterEventHandler(func(e events.RoundEnd) {
		if !matchStarted {
			return
		}
		end := EventRoundEnd{}
		if e.Winner == com.TeamCounterTerrorists {
			end.Winner = "ct"
		} else if e.Winner == com.TeamTerrorists {
			end.Winner = "t"
		}

		tick.EventRoundEnd = append(tick.EventRoundEnd, end)
		tick.Changes = true

		log.Printf("Event Round End")
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
