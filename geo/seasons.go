package geo

import (
	"log"
	"math"

	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/geoquad"
	"github.com/Flokey82/go_gens/utils"
	"github.com/davvo/mercator"
)

const (
	SeasonSpring = iota
	SeasonSummer
	SeasonAutumn
	SeasonWinter
)

const (
	SpringEquinoxDayOfYear  = 80
	SummerSolsticeDayOfYear = 172
	AutumnEquinoxDayOfYear  = 263
	WinterSolsticeDayOfYear = 355
)

// GetSeason returns the season for the current day of the year and the given
// latitude.
func (m *Geo) GetSeason(lat float64) int {
	// If we are in the northern hemisphere, the seasons are "normal".
	if lat > 0 {
		if m.GetDayOfYear() < SpringEquinoxDayOfYear {
			return SeasonWinter
		}
		if m.GetDayOfYear() < SummerSolsticeDayOfYear {
			return SeasonSpring
		}
		if m.GetDayOfYear() < AutumnEquinoxDayOfYear {
			return SeasonSummer
		}
		if m.GetDayOfYear() < WinterSolsticeDayOfYear {
			return SeasonAutumn
		}
		return SeasonWinter
	}

	// If we are in the southern hemisphere, the seasons are reversed.
	if m.GetDayOfYear() < SpringEquinoxDayOfYear {
		return SeasonSummer
	}
	if m.GetDayOfYear() < SummerSolsticeDayOfYear {
		return SeasonAutumn
	}
	if m.GetDayOfYear() < AutumnEquinoxDayOfYear {
		return SeasonWinter
	}
	if m.GetDayOfYear() < WinterSolsticeDayOfYear {
		return SeasonSpring
	}
	return SeasonSummer
}

func calculateSunPosition(latitude, longitude, altitude float64, dayOfYear int, hour float64) (elevation, azimuth float64) {
	// See: https://gml.noaa.gov/grad/solcalc/solareqns.PDF

	latRad := latitude * math.Pi / 180.0
	lonRad := longitude * math.Pi / 180.0

	// Calculate the fractional year in radians.
	// We ignore leap yers for now.
	fractionalYear := (2.0 * math.Pi / 365.0) * (float64(dayOfYear) - 1.0 + (hour / 24.0))

	// Estimate the equation of time.
	// See http://en.wikipedia.org/wiki/Equation_of_time
	eqTime := 229.18 * (0.000075 + (0.001868 * math.Cos(fractionalYear)) - (0.032077 * math.Sin(fractionalYear)) - (0.014615 * math.Cos(2.0*fractionalYear)) - (0.040849 * math.Sin(2.0*fractionalYear)))

	// Calculate the declination of the sun.
	// See http://en.wikipedia.org/wiki/Position_of_the_Sun
	declination := 0.006918 - (0.399912 * math.Cos(fractionalYear)) + (0.070257 * math.Sin(fractionalYear)) - (0.006758 * math.Cos(2.0*fractionalYear)) + (0.000907 * math.Sin(2.0*fractionalYear)) - (0.002697 * math.Cos(3.0*fractionalYear)) + (0.00148 * math.Sin(3.0*fractionalYear))

	// Calculate the timezone coarsely by longitude.
	timeZone := int(math.Floor(longitude/15.0 + 0.5))

	// Next, the true solar time is calculated in the following two equations. First the time offset is
	// found, in minutes, and then the true solar time, in minutes.
	timeOffset := eqTime + 4.0*lonRad - 60.0*float64(timeZone)

	// where eqtime is in minutes, longitude is in degrees (positive to the east of the Prime Meridian),
	// timezone is in hours from UTC (U.S. Mountain Standard Time = –7 hours).
	mn := 0.0 // minutes
	sc := 0.0 // seconds
	trueSolarTime := hour*60.0 + mn + sc/60 + timeOffset

	// The hour angle is then calculated from the true solar time.
	ha := trueSolarTime/4.0 - 180.0
	// The solar zenith angle (phi) can then be found from the hour angle (ha), latitude (lat) and solar
	// declination (decl) using the following equation:

	phi := math.Acos(math.Sin(latRad)*math.Sin(declination) + math.Cos(latRad)*math.Cos(declination)*math.Cos(ha*math.Pi/180.0))

	// The solar elevation angle (theta) can then be found from the solar zenith angle (phi) using the
	// following equation:
	theta := 90.0 - phi*180.0/math.Pi

	// The solar azimuth angle (alpha) can then be found from the hour angle (ha), latitude (lat),
	// solar declination (decl) using the following equation:
	alpha := math.Atan2(-math.Sin(ha*math.Pi/180.0), math.Cos(latRad)*math.Tan(declination)-math.Sin(latRad)*math.Cos(ha*math.Pi/180.0))

	// Now assign those values as elevation and azimuth.
	elevation = theta
	azimuth = alpha * 180.0 / math.Pi

	// Log eqation of time and declination.
	//log.Printf("Eqation of time: %f, Declination: %f", eqTime, declination*180.0/math.Pi)
	//log.Printf("Elevation: %f, Azimuth: %f", elevation, azimuth)

	return elevation, azimuth
}

func (m *Geo) GetAverageInsolation(day int) []float64 {
	useGoRoutines := true
	res := make([]float64, m.SphereMesh.NumRegions)

	chunkProcessor := func(start, end int) {
		outTri := make([]int, 0, 7)
		outRegs := make([]int, 0, 7)
		for i := start; i < end; i++ {
			// Get the base insolation value for the given latitude.
			lat := m.LatLon[i][0]
			insolation := CalcSolarRadiation(various.DegToRad(lat), day)
			//insolation := 1.0

			// Now sum up the insolation for the given day.
			var sum int
			var count int
			for hour := 0.0; hour < 24; hour += 0.25 {
				// TODO: The normal vector of the terrain and the sun vector
				// need to be taken into account. If they are not parallel,
				// the sun intensity is reduced. We could use the dot product
				// of the two vectors to calculate the intensity.
				if m.HasInsolation(i, day, hour, outTri, outRegs) {
					sum++
				}
				count++
			}
			// Calculate the average insolation for the given day.
			res[i] = insolation * float64(sum) / float64(count)
		}
	}
	if useGoRoutines {
		various.KickOffChunkWorkers(m.SphereMesh.NumRegions, chunkProcessor)
	} else {
		chunkProcessor(0, m.SphereMesh.NumRegions)
	}

	// Normalize the insolation values.
	min, max := utils.MinMax(res)
	for i := range res {
		res[i] = (res[i] - min) / (max - min)
	}
	return res
}

func latLonToPixels(lat, lon float64, zoom int) (x, y float64) {
	return mercator.LatLonToPixels(-1*lat, lon, zoom)
}

func (m *Geo) TriangulateElevation(lat, lon float64, outTri, outRegs []int) float64 {
	// Get the region at the given latitude and longitude.
	res, ok := m.RegQuadTree.FindNearestNeighbor(geoquad.Point{Lat: lat, Lon: lon})
	if !ok {
		log.Printf("lat: %f, lon: %f", lat, lon)
		panic("region not found")
	}
	zoom := 0
	if outTri == nil {
		outTri = make([]int, 0, 7)
	}
	if outRegs == nil {
		outRegs = make([]int, 0, 7)
	}
	var regs []int

	calcHeightMercator := func(p1, p2, p3, p [2]float64, z1, z2, z3 float64) float64 {
		// Convert the points to mercator.
		p1x, p1y := latLonToPixels(p1[0], p1[1], zoom)
		p2x, p2y := latLonToPixels(p2[0], p2[1], zoom)
		p3x, p3y := latLonToPixels(p3[0], p3[1], zoom)
		px, py := latLonToPixels(p[0], p[1], zoom)

		return various.CalcHeightInTriangle([2]float64{p1x, p1y}, [2]float64{p2x, p2y}, [2]float64{p3x, p3y}, [2]float64{px, py}, z1, z2, z3)
	}

	// inTriangleMercator uses the mercator projection to determine if a point is in a triangle.
	inTriangleMercator := func(p1, p2, p3, p [2]float64) bool {
		// Convert the points to mercator.
		p1x, p1y := latLonToPixels(p1[0], p1[1], zoom)
		p2x, p2y := latLonToPixels(p2[0], p2[1], zoom)
		p3x, p3y := latLonToPixels(p3[0], p3[1], zoom)
		px, py := latLonToPixels(p[0], p[1], zoom)

		// Now we can use the regular inTriangle function.
		return various.IsPointInTriangle([2]float64{p1x, p1y}, [2]float64{p2x, p2y}, [2]float64{p3x, p3y}, [2]float64{px, py})
	}

	//minDistIndex := -1
	cReg := res.Data.(int)
	latlon := [2]float64{lat, lon}
	for _, tri := range m.SphereMesh.R_circulate_t(outTri, cReg) {
		// Check if the point is inside the triangle.
		regs = m.SphereMesh.T_circulate_r(outRegs, tri)
		if inTriangleMercator(m.LatLon[regs[0]], m.LatLon[regs[1]], m.LatLon[regs[2]], latlon) {
			//minDistIndex = tri
			break
		}
	}

	return calcHeightMercator(m.LatLon[regs[0]], m.LatLon[regs[1]], m.LatLon[regs[2]], latlon, m.Elevation[regs[0]], m.Elevation[regs[1]], m.Elevation[regs[2]])
}

func wrapLat(latitude float64) float64 {
	// Ensure latitude is between -90 and 90 degrees
	if latitude > 90 {
		return 90
	} else if latitude < -90 {
		return -90
	}
	return latitude
}

func wrapLon(longitude float64) float64 {
	// Ensure longitude is between -180 and 180 degrees
	for longitude > 180 {
		longitude -= 360
	}
	for longitude < -180 {
		longitude += 360
	}
	return longitude
}

// HasInsolation returns true if the given region has insolation at the given
// time of the day and day of the year.
func (m *Geo) HasInsolation(region, day int, hour float64, outTri, outRegs []int) bool {
	// Get latitude of region.
	lat := m.LatLon[region][0]
	lon := m.LatLon[region][1]

	distSampleMultiplier := 1.0 / 10.0
	elevMultiplier := 1.0

	// Calculate sun position.
	elevation, azimuth := calculateSunPosition(lat, lon, 0, day, hour)

	// If the sun is below the horizon, there is no insolation.
	if elevation < 0 {
		return false
	}

	logDebug := false

	// Now we need to check if there is something blocking the sun in the
	// given direction.
	// So we sample the direction in 10 steps and check if there is something
	// blocking the sun.
	if logDebug {
		log.Printf("Lat: %f, Lon: %f, Elevation: %f, Azimuth: %f", lat, lon, elevation, azimuth)
	}
	// Convert elevation to radians.
	elevation = elevation * math.Pi / 180.0

	azCos := math.Cos(azimuth * math.Pi / 180.0)
	azSin := math.Sin(azimuth * math.Pi / 180.0)
	for i := 0; i < 5; i++ {
		// Calculate the latitude and longitude of the sample point.
		dist := float64(i + 1)
		dist *= distSampleMultiplier

		lat2 := lat + dist*azCos
		lon2 := lon + dist*azSin
		// Make sure lat is in the range -90 to 90 and wrap around.
		lat2 = wrapLat(lat2)
		// Make sure lon is in the range -180 to 180 and wrap around.
		lon2 = wrapLon(lon2)

		// Log the sample point.
		if logDebug {
			log.Printf("%d: Lat: %f, Lon: %f", i, lat2, lon2)
		}

		height1 := m.Elevation[region]
		if height1 < 0 {
			height1 = 0
		}

		// Get the elevation of the sample point.
		height2 := m.TriangulateElevation(lat2, lon2, outTri, outRegs)
		if height2 < 0 {
			height2 = 0
		}

		deltaHeight := height2 - height1
		if deltaHeight < 0 {
			continue
		}
		deltaHeight *= elevMultiplier

		// Calculate the distance between the two lat/lon points.
		distReg := various.Haversine(lat, lon, lat2, lon2)
		if logDebug {
			log.Printf("Dist: %f, DeltaHeight: %f", distReg, deltaHeight)
		}

		// Calculate the angle between the two regions.
		angle := math.Atan2(deltaHeight, distReg)

		// If the angle is larger than the angle of the sun, there is
		// something blocking the sun.
		if angle > elevation {
			if logDebug {
				log.Printf("Angle: %f, Elevation: %f !!!!!!!!!!!!!!!!!!", angle, elevation)
			}
			return false
		}

		// NOTE: This isn't correct or accurate, but it's a start.
		// The elevation of the regions need to be scaled since
		// tha elevation is normalized to 0-1 and the radius of the
		// sphere is 1 as well, which would be some extreme mountains.

		/*
			// Get the region at the sample point.
			res, ok := m.RegQuadTree.FindNearestNeighbor(geoquad.Point{Lat: lat2, Lon: lon2})
			if !ok {
				log.Printf("lat: %f, lon: %f", lat2, lon2)
				panic("region not found")
			}

			region2 := res.Data.(int)

			// If the region is different, we get the height of the region and
			// calculate the delta height between the two regions.
			// This and given the distance between the two regions, we can
			// calculate the angle between the two regions.
			// If the angle is larger than the angle of the sun, there is
			// something blocking the sun.
			if region2 != region {
				// Get the height of the two regions.
				height1 := m.Elevation[region]
				height2 := m.Elevation[region2]

				// Calculate the delta height.
				deltaHeight := height2 - height1
				if deltaHeight < 0 {
					continue
				}

				// Calculate the distance between the two regions.
				dist := m.GetDistance(region, region2)

				// Calculate the angle between the two regions.
				angle := math.Atan2(deltaHeight, dist)

				// If the angle is larger than the angle of the sun, there is
				// something blocking the sun.
				if angle > elevation {
					return false
				}

				// NOTE: This isn't correct or accurate, but it's a start.
				// The elevation of the regions need to be scaled since
				// tha elevation is normalized to 0-1 and the radius of the
				// sphere is 1 as well, which would be some extreme mountains.
			}
		*/
	}

	return true
}

// GetSolarRadiation returns the solar radiation for the current day of the year
// and the given latitude.
func (m *Geo) GetSolarRadiation(lat float64) float64 {
	return CalcSolarRadiation(various.DegToRad(lat), m.GetDayOfYear())
}

// calcMinMaxTemperature calculates the minimum and maximum temperature for
// every region on the current day of the year.
func (m *Geo) calcMinMaxTemperature() [][2]float64 {
	res := make([][2]float64, m.SphereMesh.NumRegions)
	for i := range res {
		lat := m.LatLon[i][0]
		res[i][0], res[i][1] = m.GetMinMaxTemperature(lat)
	}
	return res
}

// GetMinMaxTemperature returns the minimum and maximum temperature for the
// current day of the year and the given latitude.
func (m *Geo) GetMinMaxTemperature(lat float64) (min, max float64) {
	return m.GetMinMaxTemperatureOfDay(lat, m.GetDayOfYear())
}

func (m *Geo) GetMinMaxTemperatureOfDay(lat float64, dayOfYear int) (min, max float64) {
	// Get yearly average temperature for the given latitude.
	tmp := GetMeanAnnualTemp(lat)
	// TODO: Compensate for altitude.

	// Now get the average day and night duration for the given latitude.
	dayLen := calcDaylightHoursByLatitudeAndDayOfYear(various.DegToRad(lat), dayOfYear)
	nightLen := 24.0 - dayLen

	// Given the mean temperature and the day and night duration, we can
	// calculate the minimum and maximum temperature for the current day.

	// TODO: Of course the amplitude of the temperature variation is
	// dependent on the latitude. We should use a more accurate formula.
	// Also the solar radiation would influence how fast the air heats
	// up during the day. The cooling down at night should be relatively
	// constant, and only depends on the type of ground and how much
	// energy is retained by the ground.
	// TODO: Humidity is a big factor that keeps the air from cooling down
	// during the night, that's why it's so cold in the desert at night.
	min = tmp - (0.71 * nightLen)
	max = tmp + (0.71 * dayLen)
	return min, max
}

// The seasons of the year change the day/night cycle as a sine wave,
// which has an effect on the day/night temperature.
// The extremes are at the poles, where days become almost 24 hours long or
// nitghts become almost 24 hours long.
//
// At the equator, seasons are not really noticeable, as the day/night cycle
// is constant the entire year. (so the amplitude of the sine wave is 0)
//
// There is a dry and a wet season instead of the 4 seasons, which is I think
// due to the northern and southern hemispheres switching seasons and the
// global winds might push humidity or dryness across the equator. (???)
//
// The day length is almost a square wave at the poles. I think this is a
// sine wave with amplitude cap... Like a distortion guitar effect.
// The amplitude at the equator is 0, and at the poles... let's say 10, which
// would be square enough when capped at 1.
//
// We should start by calculating the daily average temperature for each region over
// a year. Then we can think about day/night temperature differences...
//
// Given this information we are able to adapt plants and animals to the seasons,
// cultures, and so on.
// http://www.atmo.arizona.edu/students/courselinks/fall16/atmo336/lectures/sec4/seasons.html
// https://github.com/woodcrafty/PyETo/blob/0b7ac9f149f4c89c5b5759a875010c521aa07f0f/pyeto/fao.py#L198 !!!
// https://github.com/willbeason/worldproc/blob/28fd3f0188082ade001110a6a73edda4b987ccdd/pkg/climate/temperature.go

func (m *Geo) CalcSolarRadiation(dayOfYear int) []float64 {
	res := make([]float64, m.SphereMesh.NumRegions)
	for i := range res {
		res[i] = CalcSolarRadiation(various.DegToRad(m.LatLon[i][0]), dayOfYear)
	}
	return res
}

// Calculate incoming solar (or shortwave) radiation, *Rs* (radiation hitting
// a horizontal plane after scattering by the atmosphere) from latitude, and
// day of year.
//
// 'latitude': Latitude [radians].
// 'dayOfYear': Day of year integer between 1 and 365 or 366).
//
// Returns incoming solar (or shortwave) radiation [MJ m-2 day-1]
func CalcSolarRadiation(latRad float64, dayOfYear int) float64 {
	daylightHours := calcDaylightHoursByLatitudeAndDayOfYear(latRad, dayOfYear)
	sunshineHours := daylightHours * 0.7 // 70% of daylight hours

	sd := solarDeclination(dayOfYear)
	sha := sunsetHourAngle(latRad, sd)
	ird := invRelDistEarthSun(dayOfYear)

	// TODO: Use clearSkyRadiation to calculate et at a given altitude.
	et := extraterrRadiation(latRad, sd, sha, ird)
	sr := solRadFromSunHours(daylightHours, sunshineHours, et)

	// At the poles, we spread the solar radiation over a wider area since
	// the angle of the sun is really flat. I just guessed that this will
	// be following 1-sin(lat-solar declination) curve.
	return sr * (1 - math.Sin(math.Abs(latRad-sd)))
}

// Calculate daylight hours from latitude and day of year.
// Based on FAO equation 34 in Allen et al (1998).
//
// 'latitude': Latitude [radians]
// 'dayOfYear': Day of year integer between 1 and 365 or 366).
//
// Returns daylight hours.
func calcDaylightHoursByLatitudeAndDayOfYear(latRad float64, dayOfYear int) float64 {
	sd := solarDeclination(dayOfYear)
	sha := sunsetHourAngle(latRad, sd)
	return daylightHours(sha)
}

// Calculate incoming solar (or shortwave) radiation, *Rs* (radiation hitting
// a horizontal plane after scattering by the atmosphere) from relative
// sunshine duration.
// If measured radiation data are not available this method is preferable
// to calculating solar radiation from temperature. If a monthly mean is
// required then divide the monthly number of sunshine hours by number of
// days in the month and ensure that *et_rad* and *daylight_hours* was
// calculated using the day of the year that corresponds to the middle of
// the month. Based on equations 34 and 35 in Allen et al (1998).
//
// 'daylightHours': Number of daylight hours [hours].
// 'sunshineHours': Sunshine duration [hours].
// 'etRad': Extraterrestrial radiation [MJ m-2 day-1].
//
// Returns incoming solar (or shortwave) radiation [MJ m-2 day-1]
func solRadFromSunHours(daylightHours, sunshineHours, etRad float64) float64 {
	// 0.5 and 0.25 are default values of regression constants (Angstrom values)
	// recommended by FAO when calibrated values are unavailable.
	epsilon := 1e-13
	return (0.5*(sunshineHours+epsilon)/(daylightHours+epsilon) + 0.25) * etRad
}

// Solar constant [ MJ m-2 min-1]
const solarConstant = 0.0820

// Calculate sunset hour angle (*Ws*) from latitude and solar
// declination. Based on FAO equation 25 in Allen et al (1998).
//
// 'latitude': Latitude [radians].
// Note: *latitude* should be negative if it in the southern
// hemisphere, positive if in the northern hemisphere.
// 'solDec': Solar declination [radians].
//
// Returns sunset hour angle [radians].
func sunsetHourAngle(latRad float64, solDec float64) float64 {
	cos_sha := -math.Tan(latRad) * math.Tan(solDec)
	// If tmp is >= 1 there is no sunset, i.e. 24 hours of daylight
	// If tmp is <= 1 there is no sunrise, i.e. 24 hours of darkness
	// See http://www.itacanet.org/the-sun-as-a-source-of-energy/
	// part-3-calculating-solar-angles/
	// Domain of acos is -1 <= x <= 1 radians (this is not mentioned in FAO-56!)
	return math.Acos(math.Min(math.Max(cos_sha, -1.0), 1.0))
}

// Calculate solar declination from day of the year.
// Based on FAO equation 24 in Allen et al (1998).
//
// 'dayOfYear': Day of year integer between 1 and 365 or 366).
//
// Returns solar declination [radians]
func solarDeclination(dayOfYear int) float64 {
	return 0.409 * math.Sin((2.0*math.Pi/365.0)*float64(dayOfYear)-1.39)
}

// Calculate daylight hours from sunset hour angle.
// Based on FAO equation 34 in Allen et al (1998).
//
// 'sha': Sunset hour angle [rad].
//
// Returns daylight hours.
func daylightHours(sha float64) float64 {
	return (24.0 / math.Pi) * sha
}

// Estimate daily extraterrestrial radiation (*Ra*, 'top of the atmosphere
// radiation').
// Based on equation 21 in Allen et al (1998). If monthly mean radiation is
// required make sure *sol_dec*. *sha* and *irl* have been calculated using
// the day of the year that corresponds to the middle of the month.
// **Note**: From Allen et al (1998): "For the winter months in latitudes
// greater than 55 degrees (N or S), the equations have limited validity.
// Reference should be made to the Smithsonian Tables to assess possible
// deviations."
//
// 'latitude': Latitude [radians]
// 'solDec': Solar declination [radians].
// 'sha': Sunset hour angle [radians].
// 'ird': Inverse relative distance earth-sun [dimensionless].
//
// Returns daily extraterrestrial radiation [MJ m-2 day-1]
func extraterrRadiation(latitude, solDec, sha, ird float64) float64 {
	tmp1 := (24.0 * 60.0) / math.Pi
	tmp2 := sha * math.Sin(latitude) * math.Sin(solDec)
	tmp3 := math.Cos(latitude) * math.Cos(solDec) * math.Sin(sha)
	return tmp1 * solarConstant * ird * (tmp2 + tmp3)
}

// Estimate clear sky radiation from altitude and extraterrestrial radiation.
// Based on equation 37 in Allen et al (1998) which is recommended when
// calibrated Angstrom values are not available.
//
// 'altitude': Elevation above sea level [m]
// 'etRad': Extraterrestrial radiation [MJ m-2 day-1].
//
// Returns clear sky radiation [MJ m-2 day-1]
func clearSkyRadiation(altitude float64, etRad float64) float64 {
	return (0.00002*altitude + 0.75) * etRad
}

// Calculate the inverse relative distance between earth and sun from
// day of the year. Based on FAO equation 23 in Allen et al (1998).
//
// 'dayOfYear': Day of the year [1 to 366]
//
// Returns inverse relative distance between earth and the sun.
func invRelDistEarthSun(dayOfYear int) float64 {
	return 1 + (0.033 * math.Cos((2.0*math.Pi/365.0)*float64(dayOfYear)))
}
