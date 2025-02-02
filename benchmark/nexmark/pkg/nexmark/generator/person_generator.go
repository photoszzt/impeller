package generator

import (
	"fmt"
	"math/rand"
	"strings"

	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/utils"
)

const (
	PERSON_ID_LEAD uint32 = 10
)

var (
	US_STATES   = [6]string{"AZ", "CA", "ID", "OR", "WA", "WY"}
	US_CITIES   = [10]string{"Phoenix", "Los Angeles", "San Francisco", "Boise", "Portland", "Bend", "Redmond", "Seattle", "Kent", "Cheyenne"}
	FIRST_NAMES = [11]string{"Peter", "Paul", "Luke", "John", "Saul", "Vicky", "Kate", "Julie", "Sarah", "Deiter", "Walter"}
	LAST_NAMES  = [9]string{"Shultz", "Abrams", "Spencer", "White", "Bartels", "Walton", "Smith", "Jones", "Noris"}
)

func NextPerson(nextEventId uint64, random *rand.Rand, timestamp int64, config *GeneratorConfig) *ntypes.Person {
	id := LastBase0PersonId(config, nextEventId) + FIRST_PERSON_ID
	name := nextPersonName(random)
	email := nextEmail(random)
	creditCard := nextCreditCard(random)
	state := nextUSState(random)
	city := nextUSCity(random)
	currentSize := 8 + len(name) + len(email) + len(creditCard) + len(city) + len(state)
	extra := NextExtra(random, uint32(currentSize), config.Configuration.AvgPersonByteSize)
	return &ntypes.Person{
		ID:           id,
		Name:         name,
		EmailAddress: email,
		CreditCard:   creditCard,
		City:         city,
		State:        state,
		DateTime:     timestamp,
		Extra:        extra,
	}
}

func NextBase0PersonId(eventId uint64, random *rand.Rand, config *GeneratorConfig) uint64 {
	numPeople := LastBase0PersonId(config, eventId) + 1
	activePeople := utils.MinUint64(numPeople, uint64(config.Configuration.NumActivePeople))
	n := NextUint64(random, activePeople+uint64(PERSON_ID_LEAD))
	return numPeople - activePeople + n
}

func LastBase0PersonId(config *GeneratorConfig, eventId uint64) uint64 {
	epoch := eventId / uint64(config.TotalProportion)
	offset := eventId % uint64(config.TotalProportion)
	if offset >= uint64(config.PersonProportion) {
		offset = uint64(config.PersonProportion) - 1
	}
	return epoch*uint64(config.Configuration.PersonProportion) + offset
}

func nextUSState(random *rand.Rand) string {
	return US_STATES[random.Intn(len(US_STATES))]
}

func nextUSCity(random *rand.Rand) string {
	return US_CITIES[random.Intn(len(US_CITIES))]
}

func nextPersonName(random *rand.Rand) string {
	return FIRST_NAMES[random.Intn(len(FIRST_NAMES))] + " " + LAST_NAMES[random.Intn(len(LAST_NAMES))]
}

func nextEmail(random *rand.Rand) string {
	return NextString(random, 7) + "@" + NextString(random, 5) + ".com"
}

func nextCreditCard(random *rand.Rand) string {
	var sb strings.Builder
	for i := 0; i < 4; i++ {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(fmt.Sprintf("%04d", random.Intn(10000)))
	}
	return sb.String()
}
