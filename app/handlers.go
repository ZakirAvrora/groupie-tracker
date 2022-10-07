package app

import (
	"Abylkaiyr/groupie-tracker/internals/filter"
	"Abylkaiyr/groupie-tracker/internals/geolocalize"
	"Abylkaiyr/groupie-tracker/internals/models"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	grabjson "Abylkaiyr/groupie-tracker/internals/grabJson"
)

func (app *AppServer) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		app.Errors(w, http.StatusNotFound, fmt.Errorf("Not Found request from %s", r.RemoteAddr))
		return
	}

	if r.Method != http.MethodGet {
		app.Errors(w, http.StatusMethodNotAllowed, fmt.Errorf("Not allowed method %v from %v",
			r.Method, r.RemoteAddr))
		return
	}

	var wg sync.WaitGroup
	var errArtists, errLocations error
	var artists []models.Artist
	var locations []string

	wg.Add(2)
	go func() {
		defer wg.Done()
		errArtists = grabjson.GetQuickArtistData(&artists)
	}()
	go func() {
		defer wg.Done()
		locations, errLocations = grabjson.GetUniqueLocations()
	}()
	wg.Wait()

	if errArtists != nil {
		app.Errors(w, http.StatusInternalServerError,
			fmt.Errorf("intenal error in getting locations: %w", errArtists))
		return
	}
	if errLocations != nil {
		app.Errors(w, http.StatusInternalServerError,
			fmt.Errorf("intenal error in getting locations: %w", errLocations))
		return
	}

	templateData := models.TemplateData{
		Artists:         artists,
		UniqueLocations: locations,
	}

	app.ParseAndExecuteTemp(w, "index.html", templateData)
}

func (app *AppServer) details(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/detail" {
		app.Errors(w, http.StatusNotFound, fmt.Errorf("Not Found request from %s", r.RemoteAddr))
		return
	}

	if r.Method != http.MethodGet {
		app.Errors(w, http.StatusMethodNotAllowed, fmt.Errorf("Invalid method from %s", r.RemoteAddr))
		return
	}

	// checking query
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		app.Errors(w, http.StatusBadRequest, fmt.Errorf("Probably inappropriate URL query from %s", r.RemoteAddr))
		return
	}

	if id < 1 || id > 52 {
		app.Errors(w, http.StatusNotFound, fmt.Errorf("no such artist with id %v", id))
	}

	var artist models.Artist

	if err := grabjson.GetDetailedData(id, &artist); err != nil {
		app.Errors(w, http.StatusInternalServerError, err)
	}
	// Locations long, latt

	for location := range artist.DatesLocations.DatesLocations {
		artist.LongLat = append(artist.LongLat,
			models.LocCoordinates(geolocalize.GetCityCoordinates(location)))
	}
	app.ParseAndExecuteTemp(w, "detail.html", artist)
}

func (app *AppServer) filters(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	creationDateFrom := r.FormValue("creationDateFrom")
	creationDateTo := r.FormValue("creationDateTo")
	firstAlbumFrom := r.FormValue("firstReleaseFrom")
	firstAlbumTo := r.FormValue("firstReleaseTo")
	filterLocation := r.FormValue("Location")
	memberNum := r.Form["members"]

	var templateData models.TemplateData
	var err error
	templateData.Artists, err = filter.FilterOut(memberNum,
		[]string{creationDateFrom, creationDateTo},
		[]string{firstAlbumFrom, firstAlbumTo},
		filterLocation)

	// if len(templateData.Artists) == 0 {
	// 	var err error = http.ErrAbortHandler
	// 	app.Errors(w, http.StatusNotFound, err)
	// 	return
	// }

	if err != nil {
		app.Errors(w, http.StatusBadRequest, err)
		return
	}

	templateData.UniqueLocations, err = grabjson.GetUniqueLocations()
	if err != nil {
		app.Errors(w, http.StatusBadRequest,
			fmt.Errorf("intenal error in getting locations: %w", err))
		return
	}

	file := "index.html"
	app.ParseAndExecuteTemp(w, file, templateData)
}
