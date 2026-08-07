package domain

// KemerovoPoints — 4 контрольные точки мониторинга по ТЗ.
// Расположены в ключевых районах города Кемерово.
// Координаты соответствуют реальным адресам установки датчиков.
var KemerovoPoints = []ForecastPoint{
	{
		ID:       "kirov",
		Name:     "Кировский район",
		Lat:      55.3909,
		Lng:      86.0683,
		District: "Кировский",
	},
	{
		ID:       "zavodsky",
		Name:     "Заводский район",
		Lat:      55.3504,
		Lng:      86.0891,
		District: "Заводский",
	},
	{
		ID:       "rudnichny",
		Name:     "Рудничный район",
		Lat:      55.4302,
		Lng:      86.0421,
		District: "Рудничный",
	},
	{
		ID:       "leninsky",
		Name:     "Ленинский район",
		Lat:      55.3648,
		Lng:      86.1098,
		District: "Ленинский",
	},
}

// KemerovoDistricts — полный список административных районов Кемерово.
var KemerovoDistricts = []string{
	"Кировский",
	"Заводский",
	"Рудничный",
	"Ленинский",
	"Центральный",
}

// PointByID возвращает точку мониторинга по ID.
// Возвращает nil, если точка не найдена.
func PointByID(id string) *ForecastPoint {
	for i := range KemerovoPoints {
		if KemerovoPoints[i].ID == id {
			return &KemerovoPoints[i]
		}
	}
	return nil
}

// PointsByDistrict возвращает точки мониторинга для заданного района.
func PointsByDistrict(district string) []ForecastPoint {
	var result []ForecastPoint
	for _, p := range KemerovoPoints {
		if p.District == district {
			result = append(result, p)
		}
	}
	return result
}
