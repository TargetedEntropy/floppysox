package main

import (
	"errors"
	"flag"
	"log"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/bot/basic"
	"github.com/Tnze/go-mc/bot/msg"
	"github.com/Tnze/go-mc/bot/screen"
	"github.com/Tnze/go-mc/bot/world"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/data/item"
	"github.com/Tnze/go-mc/level"
	GMMAuth "github.com/maxsupermanhd/go-mc-ms-auth"
)

var (
	address     = flag.String("address", "", "The server address")
	name        = flag.String("name", "Daze", "The player's name")
	playerID    = flag.String("uuid", "", "The player's UUID")
	accessToken = flag.String("token", "", "AccessToken")
)

var (
	client        *bot.Client
	player        *basic.Player
	chatHandler   *msg.Manager
	worldManager  *world.World
	screenManager *screen.Manager
)

func main() {
	mauth, err := GMMAuth.GetMCcredentials("./auth.cache", "")
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Authenticated as ", mauth.Name, " (", mauth.UUID, ")")
	client := bot.NewClient()
	client.Auth = mauth

	player = basic.NewPlayer(client, basic.DefaultSettings, basic.EventsListener{
		GameStart:    onGameStart,
		SystemMsg:    onSystemMsg,
		Disconnect:   onDisconnect,
		HealthChange: onHealthChange,
		Death:        onDeath,
	})

	chatHandler = msg.New(client, player, msg.EventsHandler{
		PlayerChatMessage: onPlayerMsg,
	})

	worldManager = world.NewWorld(client, player, world.EventsListener{
		LoadChunk:   onChunkLoad,
		UnloadChunk: onChunkUnload,
	})
	screenManager = screen.NewManager(client, screen.EventsListener{
		Open:    nil,
		SetSlot: onScreenSlotChange,
		Close:   nil,
	})

	// Login
	login_err := client.JoinServer(*address)
	if login_err != nil {
		log.Fatal(err)
	}
	log.Println("Login success")

	// JoinGame
	for {
		if err = client.HandleGame(); err == nil {
			panic("HandleGame never return nil")
		}

		if err2 := new(bot.PacketHandlerError); errors.As(err, err2) {
			if err := new(DisconnectErr); errors.As(err2, err) {
				log.Print("Disconnect: ", err.Reason)
				return
			} else {
				// print and ignore the error
				log.Print(err2)
			}
		} else {
			log.Fatal(err)
		}
	}

}

func onDeath() error {
	log.Println("Died and Respawned")
	// If we exclude Respawn(...) then the player won't press the "Respawn" button upon death
	go func() {
		time.Sleep(time.Second * 5)
		err := player.Respawn()
		if err != nil {
			log.Print(err)
		}
	}()
	return nil
}

func onGameStart() error {
	log.Println("Game start")
	if err := chatHandler.SendMessage("Hello, world"); err != nil {
		return err
	}
	return nil // if err isn't nil, HandleGame() will return it.
}

func onPlayerMsg(msg chat.Message) error {
	log.Printf("Player: %v", msg)
	return nil
}

func onSystemMsg(c chat.Message, overlay bool) error {
	log.Printf("System: %v, Overlay: %v", c, overlay)
	return nil
}

func onChunkLoad(pos level.ChunkPos) error {
	log.Println("Load chunk:", pos)
	return nil
}

func onChunkUnload(pos level.ChunkPos) error {
	log.Println("Unload chunk:", pos)
	return nil
}

func onScreenSlotChange(id, index int) error {
	if id == -2 {
		log.Printf("Slot: inventory: %v", screenManager.Inventory.Slots[index])
	} else if id == -1 && index == -1 {
		log.Printf("Slot: cursor: %v", screenManager.Cursor)
	} else {
		container, ok := screenManager.Screens[id]
		if ok {
			// Currently, only inventory container is supported
			switch container.(type) {
			case *screen.Inventory:
				slot := container.(*screen.Inventory).Slots[index]
				itemInfo := item.ByID[item.ID(slot.ID)]
				log.Printf("Slot: Screen[%d].Slot[%d]: [%v] * %d | NBT: %v", id, index, itemInfo, slot.Count, slot.NBT)
			}
		}
	}
	return nil
}

func onHealthChange(health float32) error {
	log.Printf("HealthChange: %v", health)
	return nil
}

type DisconnectErr struct {
	Reason chat.Message
}

func (d DisconnectErr) Error() string {
	return "disconnect: " + d.Reason.String()
}

func onDisconnect(reason chat.Message) error {
	// return an error value so that we can stop main loop
	return DisconnectErr{Reason: reason}
}
