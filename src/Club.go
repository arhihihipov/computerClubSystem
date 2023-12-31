package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Club struct {
	lines            []string              // Команды из файла
	output           []string              // Выходная информация
	clients          []string              // Список клиентов, которые находятся в клубе
	waitingList      []string              // Очередь ожидания
	cost             int                   // Стоимость часа в клубе
	tables           int                   // Число столов в клубе
	open             time.Time             // Время открытия клуба
	close            time.Time             // Время закрытия клуба
	tableTimeTracker map[int]time.Duration // Время занятости каждого стола
	tableOccupancy   map[int]Ticket        // Кто за каким столом сидит
	proceeds         map[int]int           // Выручка с каждого стола
}

// Клиент уходит из клуба. Возвращаемое значение -- номер стола, который он освободил
func (c *Club) clientLeave(clientName string, leavingTime time.Time) int {
	curTable := -1
	for key, ticket := range c.tableOccupancy {
		if ticket.name == clientName {
			curTable = key
		}
	}

	if curTable != -1 {
		// Подбиваем прибыль, если клиент перед уходом сидел за столом
		spentTime := leavingTime.Sub(c.tableOccupancy[curTable].startTime)
		c.tableTimeTracker[curTable] += spentTime
		c.proceeds[curTable] += int(math.Ceil(leavingTime.Sub(c.tableOccupancy[curTable].startTime).Hours())) * c.cost

		delete(c.tableOccupancy, curTable)
	}

	// Удаляем клиента из списка посетителей
	idx := -1
	for i := range c.clients {
		if c.clients[i] == clientName {
			idx = i
			break
		}
	}
	if idx != -1 {
		c.clients = append(c.clients[:idx], c.clients[idx+1:]...)
	}

	// Удаляем клиента из листа ожидания
	idx = -1
	for i := range c.waitingList {
		if c.waitingList[i] == clientName {
			idx = i
			break
		}
	}
	if idx != -1 {
		c.waitingList = append(c.waitingList[:idx], c.waitingList[idx+1:]...)
	}
	// Возвращаем номер стола, который освободил клиент. Если этот клиент не сидел за столом, то возвращаем -1
	return curTable
}

// Проверка, есть ли клиент с таким именем в клубе
func (c *Club) isClientHere(clientName string) bool {
	for _, client := range c.clients {
		if client == clientName {
			return true
		}
	}
	return false
}

// Заполнение полей структуры
func (c *Club) init(data []string) {
	c.lines = data
	// Получаем информацию о количестве столов
	c.tables, _ = strconv.Atoi(data[0])

	c.tableTimeTracker = make(map[int]time.Duration)
	c.tableOccupancy = make(map[int]Ticket)
	c.proceeds = make(map[int]int)

	for i := 1; i <= c.tables; i++ {
		c.tableTimeTracker[i] = 0
		c.proceeds[i] = 0
	}

	// Получаем информацию о времени работы
	openCloseTime := strings.Split(data[1], " ")
	c.open, _ = time.Parse("15:04", openCloseTime[0])
	c.close, _ = time.Parse("15:04", openCloseTime[1])

	// получаем информацию о стоимости часа в клубе
	c.cost, _ = strconv.Atoi(data[2])
}

// Обработка строк файла
func (c *Club) processing() {
	c.output = append(c.output, c.open.Format("15:04"))

	for i := 3; i < len(c.lines); i++ {
		c.output = append(c.output, c.lines[i])

		event := strings.Split(c.lines[i], " ")
		eventTime, _ := time.Parse("15:04", event[0])
		eventID, _ := strconv.Atoi(event[1])
		clientName := event[2]

		switch eventID {

		case 1:
			// Проверяем, не заходил ли клиент в клуб ранее
			if c.isClientHere(clientName) {
				c.output = append(c.output, makeEvent(eventTime, 13, "YouShallNotPass"))
				break
			}
			// Клиент пришел не во время работы клуба
			if eventTime.Before(c.open) || eventTime.After(c.close) {
				c.output = append(c.output, makeEvent(eventTime, 13, "NotOpenYet"))
				break
			}

			c.clients = append(c.clients, clientName)

		case 2:
			table, _ := strconv.Atoi(event[3])

			// Клиент на данный момент не в клубе
			if !c.isClientHere(clientName) {
				c.output = append(c.output, makeEvent(eventTime, 13, "ClientUnknown"))
				break
			}

			// Стол, за который хочет пересесть клиент, занят
			if _, found := c.tableOccupancy[table]; found {
				c.output = append(c.output, makeEvent(eventTime, 13, "PlaceIsBusy"))
				break
			}

			// Если посетитель пересаживается, то oldTable хранит в себе номер стола, с которого посетитель хочет пересесть
			// Если посетитель не пересаживается, то oldTable = -1
			oldTable := -1
			for key, ticket := range c.tableOccupancy {
				if ticket.name == clientName {
					oldTable = key
				}
			}

			if oldTable != -1 {
				// Клиент пересаживается, подбиваем выручку за его столом и удаляем тикет
				spentTime := eventTime.Sub(c.tableOccupancy[oldTable].startTime)
				c.tableTimeTracker[oldTable] += spentTime
				c.proceeds[oldTable] += int(math.Ceil(eventTime.Sub(c.tableOccupancy[oldTable].startTime).Hours())) * c.cost

				delete(c.tableOccupancy, oldTable)
			}
			// Сажаем клиента за новый стол
			c.tableOccupancy[table] = Ticket{clientName, eventTime}

		case 3:
			// Клиент на данный момент не в клубе
			if !c.isClientHere(clientName) {
				c.output = append(c.output, makeEvent(eventTime, 13, "ClientUnknown"))
				break
			}

			if len(c.tableOccupancy) < c.tables {
				c.output = append(c.output, makeEvent(eventTime, 13, "ICanWaitNoLonger!"))
				break
			}

			if len(c.waitingList) > c.tables {
				c.output = append(c.output, makeEvent(eventTime, 11, clientName))
				c.clientLeave(clientName, eventTime)
				break
			}

			c.waitingList = append(c.waitingList, clientName)

		case 4:
			if !c.isClientHere(clientName) {
				c.output = append(c.output, makeEvent(eventTime, 13, "ClientUnknown"))
				break
			}

			freePlace := c.clientLeave(clientName, eventTime)

			// Если ушедший клиент освободил место, то сажаем за него первого в листе ожидания
			if freePlace != -1 {
				if len(c.waitingList) >= 1 {
					client := c.waitingList[0]
					c.waitingList = c.waitingList[1:]
					c.tableOccupancy[freePlace] = Ticket{client, eventTime}
					c.output = append(c.output, makeEvent(eventTime, 12, fmt.Sprintf("%s %d", client, freePlace)))
				}
			}

		}
	}

	// Сортируем оставшихся клиентов в алфавитном порядке и выдворяем их из клуба
	sort.Strings(c.clients)
	length := len(c.clients)
	for i := 0; i < length; i++ {
		client := c.clients[0]
		c.clientLeave(client, c.close)
		c.output = append(c.output, makeEvent(c.close, 11, client))
	}

	// Клуб закрывается
	c.output = append(c.output, c.close.Format("15:04"))

	// Выводим информацию о столах
	for i := 1; i <= c.tables; i++ {
		tableTime, _ := time.Parse("15:04", "00:00")
		tableInfo := fmt.Sprintf("%d %d %s", i, c.proceeds[i], tableTime.Add(c.tableTimeTracker[i]).Format("15:04"))
		c.output = append(c.output, tableInfo)
	}
}
