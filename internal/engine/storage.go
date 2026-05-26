package engine

import "log"

func (e *UptimeEngine) getTargetByUrl(url string) int {
	rows, err := e.GetTargetByUrlStatement.Query(url)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	id := -1
	var targetURL string
	var createdAt string
	for rows.Next() {
		if err := rows.Scan(&id, &targetURL, &createdAt); err != nil {
			log.Printf("Did not find target due to %s", err)
			return id
		}
		log.Printf("Found Target in DB - ID: %d, URL: %s, created at: %s", id, targetURL, createdAt)
	}

	return id
}

func (e *UptimeEngine) saveTarget(url string) int {
	res, err := e.SaveTargetStatement.Exec(url)
	if err != nil {
		log.Fatal(err)
	}
	newId, _ := res.LastInsertId()
	return int(newId)
}

func (e *UptimeEngine) saveResult(result Result) {
	// Check if URL has been pinged before
	url := result.URL
	targetId := e.getTargetByUrl(url)
	if targetId == -1 {
		targetId = e.saveTarget(url)
	}

	statusCode := result.StatusCode
	latencyMs := result.Latency
	var errMessage string
	if statusCode == -1 {
		errMessage = result.Error.Error()

	}
	if _, err := e.SavePingStatement.Exec(targetId, statusCode, errMessage, latencyMs); err != nil {
		log.Fatal(err)
	}
}

func (e *UptimeEngine) getAllTargetUrls() []string {
	urlList := []string{}
	rows, err := e.GetAllTargetUrlsStatement.Query()
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			log.Printf("Could not process URL due to %s", err)
		}
		urlList = append(urlList, url)
	}

	return urlList
}
