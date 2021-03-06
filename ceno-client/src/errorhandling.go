package main

import (
	"bytes"
	"encoding/json"
	"github.com/nicksnyder/go-i18n/i18n"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
)

// CC errors
const (
	ERR_NO_CONFIG              = 1100
	ERR_MALFORMED_URL          = 1101
	ERR_MISSING_VIEW           = 1102
	ERR_NO_FEEDS_FILE          = 1103
	ERR_NO_ARTICLES_FILE       = 1104
	ERR_CORRUPT_JSON           = 1105
	ERR_MALFORMED_STATUS_CHECK = 1106
	ERR_NO_CONNECT_LCS         = 1200
	ERR_MALFORMED_LCS_RESPONSE = 1201
	ERR_FROM_LCS               = 1202
	ERR_NO_CONNECT_RS          = 1203
	ERR_LCS_NOT_READY          = 1204
	ERR_INVALID_ERROR          = 100
)

// LCS errors that can be reported to the CC
const (
	ERR_LCS_MALFORMED_URL  = 2110
	ERR_LCS_URL_DECODE     = 2112
	ERR_LCS_WILL_NOT_SERVE = 2120
	ERR_LCS_LOOKUP_FAILURE = 2130
	ERR_LCS_INTERNAL       = 2140
	ERR_LCS_WAIT_FREENET   = 2300
	ERR_LCS_WAIT_PEERS     = 2301
)

// An email that can be displayed on pages served by the CC to email CENO devs about a problem
const contactInfo = "ceno@equalit.ie"

// Types for enumerating, storing state surrounding, and handling errors
type ErrorCode uint32
type ErrorState map[string]interface{}
type ErrorHandler func(ErrorState) bool

/******************************************************************************************
 ************************************ PRIVATE VALUES **************************************
 ******************************************************************************************/

// Map error codes to the ids of explanations of errors that are localizable
var errorAdvice = map[ErrorCode]string{
	ERR_NO_CONFIG:              "missing_config_err",
	ERR_MALFORMED_URL:          "malformed_url_err",
	ERR_NO_FEEDS_FILE:          "no_feeds_file_err",
	ERR_NO_ARTICLES_FILE:       "no_articles_advice",
	ERR_CORRUPT_JSON:           "corrupt_json_err",
	ERR_NO_CONNECT_LCS:         "agent_communication_err",
	ERR_MALFORMED_LCS_RESPONSE: "contact_devs_err",
	ERR_FROM_LCS:               "",
	ERR_NO_CONNECT_RS:          "agent_communication_err",
	ERR_MISSING_VIEW:           "download_package_err",
	ERR_INVALID_ERROR:          "contact_devs_err",
	ERR_LCS_NOT_READY:          "lcs_not_ready_err",
	ERR_LCS_MALFORMED_URL:      "malformed_url_err",
	ERR_LCS_URL_DECODE:         "malformed_url_err",
	ERR_LCS_WILL_NOT_SERVE:     "malformed_url_err",
	ERR_LCS_LOOKUP_FAILURE:     "lcs_lookup_failure_err",
	ERR_LCS_INTERNAL:           "lcs_lookup_failure_err",
	ERR_LCS_WAIT_FREENET:       "lcs_lookup_failure_err",
	ERR_LCS_WAIT_PEERS:         "lcs_lookup_failure_err",
}

// An error handler for each of the errors that CC is expected to be responsible for.
// Information about the state of the program during the time the error occurred, required
// for the error to be handled, should be encoded into the ErrorState map.
var ccErrorHandlers = map[ErrorCode]func(ErrorState) bool{
	ERR_NO_CONFIG:              downloadConfigAndServeError,
	ERR_MALFORMED_URL:          serveError,
	ERR_NO_FEEDS_FILE:          downloadFeedsFileAndServeError,
	ERR_NO_ARTICLES_FILE:       downloadArticlesFileAndServeError,
	ERR_CORRUPT_JSON:           serveError,
	ERR_NO_CONNECT_LCS:         serveError,
	ERR_MALFORMED_LCS_RESPONSE: ReportDecodeError,
	ERR_FROM_LCS:               handleLCSErrorReport,
	ERR_NO_CONNECT_RS:          serveError,
	ERR_MISSING_VIEW:           downloadViewsAndServeError,
	ERR_INVALID_ERROR:          serveError,
	ERR_LCS_NOT_READY:          serveError,
}

// An error handler for each of the error thatthe LCS is expected to send to the
// CC for handling.  Information about the state of the program during the time
// the error occurred should be encoded in the ErrorState map.
var lcsErrorHandlers = map[ErrorCode]func(ErrorState) bool{
	ERR_LCS_MALFORMED_URL:  serveError,
	ERR_LCS_URL_DECODE:     serveError,
	ERR_LCS_WILL_NOT_SERVE: serveError,
	ERR_LCS_LOOKUP_FAILURE: serveError,
	ERR_LCS_INTERNAL:       serveError,
	ERR_LCS_WAIT_FREENET:   showFreenetMonitorAndServeError,
	ERR_LCS_WAIT_PEERS:     showPeerMonitorAndServeError,
}

// Some errors will resolve themselves over time, and so the error page should,
// depending on the error being served, automatically refresh itself the same way
// that wait.html does.
var AutoRefreshingErrorPages = map[ErrorCode]bool{
	ERR_NO_FEEDS_FILE:          true,
	ERR_NO_ARTICLES_FILE:       true,
	ERR_NO_CONNECT_LCS:         true,
	ERR_MALFORMED_LCS_RESPONSE: true,
	ERR_FROM_LCS:               true,
	ERR_NO_CONNECT_RS:          true,
	ERR_LCS_LOOKUP_FAILURE:     true,
	ERR_LCS_INTERNAL:           true,
	ERR_LCS_WAIT_FREENET:       true,
}

/********************
 ** ERROR HANDLERS **
 ********************/

/**
 * Prepare and serve the standard error page with relevant information.
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page was served successfully
 */
func serveError(state ErrorState) bool {
	w := state["responseWriter"].(http.ResponseWriter)
	r := state["request"].(*http.Request)
	errMsg := state["errMsg"].(string)
	errCode := state["errCode"].(ErrorCode)
	ExecuteErrorPage(errCode, errMsg, w, r)
	return true
}

// We have a number of placeholder functions that could be expanded on to have the CC
// respond to certain kinds of errors by doing useful things in the background after
// simply serving an error page.
// Due to time constraints, we haven't implemented any of these.  No critical functionality
// depends on them.

/**
 * Download the default configuration file package, validate, and apply before serving an error page
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func downloadConfigAndServeError(state ErrorState) bool {
	// temporary
	return serveError(state)
}

/**
 * Download and save the latest RSS feeds list file and serve an error saying this is happening.
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func downloadFeedsFileAndServeError(state ErrorState) bool {
	// temporary
	return serveError(state)
}

/**
 * Download and save an articles list file and serve an error saying this is happening.
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func downloadArticlesFileAndServeError(state ErrorState) bool {
	// temporary
	return serveError(state)
}

/**
 * Download the default configuration file package, validate, and apply before serving an error page
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func handleLCSErrorReport(state ErrorState) bool {
	// temporary
	return serveError(state)
}

/**
 * Download the default configuration file package, validate, and apply before serving an error page
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func downloadViewsAndServeError(state ErrorState) bool {
	// temporary
	return serveError(state)
}

/**
 * Download the default configuration file package, validate, and apply before serving an error page
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func showFreenetMonitorAndServeError(state ErrorState) bool {
	// temporary
	return serveError(state)
}

/**
 * Download the default configuration file package, validate, and apply before serving an error page
 * @param {ErrorState} state - Must contain HTTP request and response objects and error message
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func showPeerMonitorAndServeError(state ErrorState) bool {
	// temporary
	return serveError(state)
}

/*****************************************************************************************
 ************************************ PUBLIC VALUES **************************************
 *****************************************************************************************/

/**
 * Handle errors occurring in the CC. This function terminates requests.
 * @param {ErrorCode} errCode - The error code identifying the error that occurred
 * @param {string} errMsg - A message to output with the error page, if any
 * @param {ErrorState} state - State information about the program at the time the error was returned
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func HandleCCError(errCode ErrorCode, errMsg string, state ErrorState) bool {
	if _, hasErrorCode := state["errCode"]; !hasErrorCode {
		state["errCode"] = errCode
	}
	if _, hasErrorMsg := state["errMsg"]; !hasErrorMsg {
		state["errMsg"] = errMsg
	}
	return ccErrorHandlers[errCode](state)
}

/**
 * Handle errors reported by the LCS.  This function should terminate requests.
 * @param {ErrorCode} errCode - The error code identifying the error that occurred
 * @param {string} errMsg - A message to output with the error page, if any
 * @param {ErrorState} state - State information about the program at the time the error was returned
 * @return true if the error page is served successfully and other background tasks completed okay
 */
func HandleLCSError(errCode ErrorCode, errMsg string, state ErrorState) bool {
	if _, hasErrorCode := state["errCode"]; !hasErrorCode {
		state["errCode"] = errCode
	}
	if _, hasErrorMsg := state["errMsg"]; !hasErrorMsg {
		state["errMsg"] = errMsg
	}
	return lcsErrorHandlers[errCode](state)
}

/**
 * Report that an error occurred trying to decode the response from the LCS.
 * @param {ErrorState} state - Must contain error message to send and the URL to send the request to
 * @return true if the error request was sent successfully
 */
func ReportDecodeError(state ErrorState) bool {
	mapping := map[string]interface{}{
		"error": state["errMsg"].(string),
	}
	marshalled, _ := json.Marshal(mapping)
	reader := bytes.NewReader(marshalled)
	req, err := http.NewRequest("POST", state["reportURL"].(string), reader)
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(req)
	return response.StatusCode == 200
}

/**
 * Execute the error template or produce a helpful plaintext response to explain
 * the error and provide pre-composed advice.
 * @param {ErrorCode} errorCode - The code number identifying the error that occurred
 * @param {string} errorMsg - A message to go along with the error report
 * @param {ResponseWriter} w - The object handling responding to the client
 * @param {*Request} r - Information about the request
 */
func ExecuteErrorPage(errorCode ErrorCode, errorMsg string, w http.ResponseWriter, r *http.Request) {
	T, _ := i18n.Tfunc(os.Getenv("CENOLANG"), "en-us")
	t, err := template.ParseFiles(path.Join(".", "views", "error.html"))
	advice, foundErr := errorAdvice[errorCode]
	if !foundErr {
		errMsg := T("unrecognized_error_code", map[string]interface{}{"ErrCode": errorCode})
		ExecuteErrorPage(ERR_INVALID_ERROR, errMsg, w, r)
	} else if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(T("missing_view", map[string]interface{}{"View": "error.html"})))
	} else {
		shouldRefresh := AutoRefreshingErrorPages[errorCode]
		t.Execute(w, map[string]string{
			"Url":              r.URL.String(),
			"Error":            errorMsg,
			"ShouldRefresh":    strconv.FormatBool(shouldRefresh),
			"Advice":           T(advice),
			"NoBundlePrepared": T("no_bundle_prepared_html"),
			"YouAskedFor":      T("you_asked_for_html"),
			"ErrorWeGot":       T("error_we_got_html"),
			"WhatYouCanDo":     T("what_you_can_do_html"),
			"Retry":            T("retry_html"),
			"Report":           T("report_html"),
		})
	}
}

/**
 * Determine whether an error code is one internal to the CC.
 * This is the case when it is of the form 1XXX.
 * @param {ErrorCode} errorCode - The code number identifying the error.
 * @return true if the error code provided belongs to the CC, else false
 */
func IsClientError(errorCode ErrorCode) bool {
	return errorCode/1000 == 1
}

/**
 * Determine whether an error code is one sent from the LCS.
 * This is the case when it is of the form 2YYYY.
 * @param {ErrorCode} errorCode - The code number identifying the error.
 * @return true if the error code provided belongs to the LCS, else false
 */
func IsCacheServerError(errorCode ErrorCode) bool {
	return errorCode/1000 == 2
}
