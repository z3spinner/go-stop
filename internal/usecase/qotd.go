package usecase

import "time"

// Quote of the day for the test push — eco / left / revolutionary / nature
// themed. Kept server-side (not client-supplied) so a caller can never author
// the notification content; the client only chooses a language. author is shared
// across languages (proper names).
type qotd struct {
	author string
	text   map[string]string // locale -> text
}

var qotdTitle = map[string]string{
	"fr": "Citation du jour",
	"en": "Quote of the day",
	"de": "Zitat des Tages",
	"es": "Cita del día",
	"it": "Citazione del giorno",
	"nl": "Citaat van de dag",
}

var qotdQuotes = []qotd{
	{author: "Karl Marx", text: map[string]string{
		"fr": "De chacun selon ses moyens, à chacun selon ses besoins.",
		"en": "From each according to his ability, to each according to his needs.",
		"de": "Jeder nach seinen Fähigkeiten, jedem nach seinen Bedürfnissen.",
		"es": "De cada cual según su capacidad, a cada cual según sus necesidades.",
		"it": "Da ciascuno secondo le sue capacità, a ciascuno secondo i suoi bisogni.",
		"nl": "Van ieder naar vermogen, aan ieder naar behoefte.",
	}},
	{author: "Karl Marx", text: map[string]string{
		"fr": "Les philosophes n'ont fait qu'interpréter le monde, il s'agit maintenant de le transformer.",
		"en": "The philosophers have only interpreted the world; the point is to change it.",
		"de": "Die Philosophen haben die Welt nur verschieden interpretiert; es kommt aber darauf an, sie zu verändern.",
		"es": "Los filósofos no han hecho más que interpretar el mundo; de lo que se trata es de transformarlo.",
		"it": "I filosofi hanno soltanto interpretato il mondo; ora si tratta di trasformarlo.",
		"nl": "De filosofen hebben de wereld slechts geïnterpreteerd; het komt erop aan haar te veranderen.",
	}},
	{author: "Marx & Engels", text: map[string]string{
		"fr": "Prolétaires de tous les pays, unissez-vous !",
		"en": "Workers of the world, unite!",
		"de": "Proletarier aller Länder, vereinigt euch!",
		"es": "¡Proletarios de todos los países, uníos!",
		"it": "Proletari di tutti i paesi, unitevi!",
		"nl": "Proletariërs aller landen, verenigt u!",
	}},
	{text: map[string]string{
		"fr": "Soyez réalistes, demandez l'impossible.",
		"en": "Be realistic, demand the impossible.",
		"de": "Seid realistisch, verlangt das Unmögliche.",
		"es": "Seamos realistas, pidamos lo imposible.",
		"it": "Siate realisti, chiedete l'impossibile.",
		"nl": "Wees realistisch, eis het onmogelijke.",
	}},
	{text: map[string]string{
		"fr": "Le peuple uni ne sera jamais vaincu.",
		"en": "The people united will never be defeated.",
		"de": "Ein vereintes Volk wird niemals besiegt.",
		"es": "El pueblo unido jamás será vencido.",
		"it": "Il popolo unito non sarà mai vinto.",
		"nl": "Een verenigd volk zal nooit worden verslagen.",
	}},
	{author: "Che Guevara", text: map[string]string{
		"fr": "La solidarité est la tendresse des peuples.",
		"en": "Solidarity is the tenderness of the peoples.",
		"de": "Solidarität ist die Zärtlichkeit der Völker.",
		"es": "La solidaridad es la ternura de los pueblos.",
		"it": "La solidarietà è la tenerezza dei popoli.",
		"nl": "Solidariteit is de tederheid van de volkeren.",
	}},
	{text: map[string]string{
		"fr": "Un autre monde est possible.",
		"en": "Another world is possible.",
		"de": "Eine andere Welt ist möglich.",
		"es": "Otro mundo es posible.",
		"it": "Un altro mondo è possibile.",
		"nl": "Een andere wereld is mogelijk.",
	}},
	{author: "Proudhon", text: map[string]string{
		"fr": "La propriété, c'est le vol.",
		"en": "Property is theft.",
		"de": "Eigentum ist Diebstahl.",
		"es": "La propiedad es un robo.",
		"it": "La proprietà è un furto.",
		"nl": "Eigendom is diefstal.",
	}},
	{author: "Frederick Douglass", text: map[string]string{
		"fr": "Sans lutte, il n'y a pas de progrès.",
		"en": "Without struggle, there is no progress.",
		"de": "Ohne Kampf kein Fortschritt.",
		"es": "Sin lucha no hay progreso.",
		"it": "Senza lotta non c'è progresso.",
		"nl": "Zonder strijd is er geen vooruitgang.",
	}},
	{author: "Chief Seattle", text: map[string]string{
		"fr": "La Terre ne nous appartient pas, c'est nous qui appartenons à la Terre.",
		"en": "The Earth does not belong to us; we belong to the Earth.",
		"de": "Die Erde gehört nicht uns; wir gehören der Erde.",
		"es": "La Tierra no nos pertenece; nosotros pertenecemos a la Tierra.",
		"it": "La Terra non appartiene a noi; siamo noi ad appartenere alla Terra.",
		"nl": "De aarde behoort niet ons toe; wij behoren de aarde toe.",
	}},
	{text: map[string]string{
		"fr": "Quand le dernier arbre sera abattu, nous comprendrons que l'argent ne se mange pas.",
		"en": "Only when the last tree is cut down will we realize we cannot eat money.",
		"de": "Erst wenn der letzte Baum gefällt ist, werden wir merken, dass man Geld nicht essen kann.",
		"es": "Solo cuando el último árbol sea talado comprenderemos que el dinero no se come.",
		"it": "Solo quando l'ultimo albero sarà abbattuto capiremo che il denaro non si mangia.",
		"nl": "Pas als de laatste boom geveld is, beseffen we dat we geld niet kunnen eten.",
	}},
	{author: "Rachel Carson", text: map[string]string{
		"fr": "Dans la nature, rien n'existe seul.",
		"en": "In nature nothing exists alone.",
		"de": "In der Natur existiert nichts für sich allein.",
		"es": "En la naturaleza nada existe en soledad.",
		"it": "In natura nulla esiste da solo.",
		"nl": "In de natuur bestaat niets op zichzelf.",
	}},
	{text: map[string]string{
		"fr": "Nous n'héritons pas de la Terre de nos ancêtres, nous l'empruntons à nos enfants.",
		"en": "We do not inherit the Earth from our ancestors; we borrow it from our children.",
		"de": "Wir erben die Erde nicht von unseren Vorfahren, wir borgen sie von unseren Kindern.",
		"es": "No heredamos la Tierra de nuestros antepasados, la tomamos prestada de nuestros hijos.",
		"it": "Non ereditiamo la Terra dai nostri antenati, la prendiamo in prestito dai nostri figli.",
		"nl": "We erven de aarde niet van onze voorouders, we lenen haar van onze kinderen.",
	}},
	{author: "Emiliano Zapata", text: map[string]string{
		"fr": "Mieux vaut mourir debout que vivre à genoux.",
		"en": "I would rather die on my feet than live on my knees.",
		"de": "Lieber aufrecht sterben als auf den Knien leben.",
		"es": "Prefiero morir de pie que vivir de rodillas.",
		"it": "Meglio morire in piedi che vivere in ginocchio.",
		"nl": "Liever staand sterven dan op je knieën leven.",
	}},
	{text: map[string]string{
		"fr": "La terre et la liberté.",
		"en": "Land and freedom.",
		"de": "Land und Freiheit.",
		"es": "Tierra y libertad.",
		"it": "Terra e libertà.",
		"nl": "Land en vrijheid.",
	}},
	{text: map[string]string{
		"fr": "Vis simplement pour que d'autres puissent simplement vivre.",
		"en": "Live simply so that others may simply live.",
		"de": "Lebe einfach, damit andere einfach leben können.",
		"es": "Vive con sencillez para que otros, sencillamente, puedan vivir.",
		"it": "Vivi con semplicità affinché altri possano semplicemente vivere.",
		"nl": "Leef eenvoudig, zodat anderen eenvoudig kunnen leven.",
	}},
	{author: "Rosa Luxemburg", text: map[string]string{
		"fr": "La liberté est toujours la liberté de celui qui pense autrement.",
		"en": "Freedom is always the freedom of those who think differently.",
		"de": "Freiheit ist immer die Freiheit des Andersdenkenden.",
		"es": "La libertad es siempre la libertad del que piensa diferente.",
		"it": "La libertà è sempre la libertà di chi la pensa diversamente.",
		"nl": "Vrijheid is altijd de vrijheid van wie anders denkt.",
	}},
	{author: "Eduardo Galeano", text: map[string]string{
		"fr": "Beaucoup de petites gens, dans de petits endroits, faisant de petites choses, peuvent changer le monde.",
		"en": "Many small people, in small places, doing small things, can change the world.",
		"de": "Viele kleine Leute, an vielen kleinen Orten, die viele kleine Dinge tun, können die Welt verändern.",
		"es": "Mucha gente pequeña, en lugares pequeños, haciendo cosas pequeñas, puede cambiar el mundo.",
		"it": "Molte piccole persone, in piccoli luoghi, facendo piccole cose, possono cambiare il mondo.",
		"nl": "Veel kleine mensen, op kleine plekken, die kleine dingen doen, kunnen de wereld veranderen.",
	}},
	{author: "Buenaventura Durruti", text: map[string]string{
		"fr": "Nous portons un monde nouveau dans nos cœurs.",
		"en": "We carry a new world here, in our hearts.",
		"de": "Wir tragen eine neue Welt in unseren Herzen.",
		"es": "Llevamos un mundo nuevo en nuestros corazones.",
		"it": "Portiamo un mondo nuovo nei nostri cuori.",
		"nl": "Wij dragen een nieuwe wereld in ons hart.",
	}},
	{author: "Howard Zinn", text: map[string]string{
		"fr": "De petits gestes, multipliés par des millions de personnes, peuvent transformer le monde.",
		"en": "Small acts, when multiplied by millions of people, can transform the world.",
		"de": "Kleine Taten, millionenfach vervielfacht, können die Welt verändern.",
		"es": "Pequeños actos, multiplicados por millones de personas, pueden transformar el mundo.",
		"it": "Piccoli gesti, moltiplicati per milioni di persone, possono trasformare il mondo.",
		"nl": "Kleine daden, vermenigvuldigd met miljoenen mensen, kunnen de wereld veranderen.",
	}},
}

func pick(m map[string]string, lang string) string {
	if v, ok := m[lang]; ok && v != "" {
		return v
	}
	return m["fr"]
}

// quoteOfTheDay returns the day's localized title + body. A different quote each
// day (UTC), cycling through all of them. `now` is injectable for tests.
func quoteOfTheDay(lang string, now time.Time) (title, body string) {
	day := int(now.UTC().Unix() / 86400)
	q := qotdQuotes[((day%len(qotdQuotes))+len(qotdQuotes))%len(qotdQuotes)]
	text := pick(q.text, lang)
	if q.author != "" {
		text += " — " + q.author
	}
	return pick(qotdTitle, lang), text
}
