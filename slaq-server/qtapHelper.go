package main

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func getIcsUrlFromUsernameAndPassword(username string, password string) (string, error) {
	// We need a cookie jar to set a cookie for a request... That's why we are doing this
	// nil represents custom options, which we don't want/care about
	cookieJar, err := cookiejar.New(nil)
	// This will never throw an error
	if err != nil {
		fmt.Println("Error creating cookiejar: ", err)
		return "", errors.New("Error creating cookiejar")
	}

	// Slice of cookies that we will eventually give to our cookie jar
	// Why we can't associate a slice with our request, bypassing
	// the cookie jar, is beyond me
	var cookies []*http.Cookie

	// The magic QTap cookie!
	oneCookie := &http.Cookie{
		Name:   "BIGipServerrp_my_queensu",
		Value:  "621151242.20480.0000",
		Domain: "my.queensu.ca",
		// Path means this and all subdirectories
		// so this cookie is valid for
		// my.queensu.ca/*
		Path: "/",
	}

	// Add it to our cookie slice
	cookies = append(cookies, oneCookie)

	// Parse this hardcoded string into a URL
	// This will never fail because the URL is a constant.
	targetUrl, err := url.Parse("https://my.queensu.ca/software-centre")
	if err != nil {
		fmt.Println("Error parsing constant URL: ", err)
		return "", errors.New("Error parsing constant URL")
	}

	// Point the cookie jar to our slice of cookies
	cookieJar.SetCookies(targetUrl, cookies)

	// Create a custom HTTP client that uses our cookie jar
	client := &http.Client{
		Jar: cookieJar,
	}

	// We make a request to set up a login session
	// We get some cookies associated with our client as a result

	// We are making use of the default redirect behaviour - follow <= 10 redirects
	resp, err := client.Get("https://my.queensu.ca/software-centre")
	if err != nil {
		fmt.Println("Error getting page: ", err)
		return "", errors.New("Error getting page")
	}

	// Now that we have cookies, we need to POST our actual login credentials
	postValues := url.Values{}
	postValues.Add("j_username", username)
	postValues.Add("j_password", password)
	// Note the weirdness about this value
	// The key is Login, and the Value is <space>Log In<space>
	postValues.Add("Login", " Log In ")

	// We are going to replace the resp, so let's be good citizens and close the old one
	resp.Body.Close()

	// Now perform the login
	resp, err = client.PostForm("https://login.queensu.ca/idp/Authn/UserPassword", postValues)

	// This login gives us an HTTP 200 response, but wants us to redirect in Javascript by submitting
	// a hidden form. Yikes. So we need to extract the form information from the page.

	// These are the two POST data points we will need
	var relayState, SamlResponse html.Token

	// Programmatically access the DOM
	d := html.NewTokenizer(resp.Body)
	for {

		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			break
		}
		token := d.Token()
		switch tokenType {
		// Self closing tag, like <input /> which is what we are looking for
		case html.SelfClosingTagToken:
			// Only if the tag is actually an input tag
			if token.DataAtom == atom.Input {
				for _, v := range token.Attr {
					// Look for our two desired values
					if v.Key == "name" && v.Val == "RelayState" {
						relayState = token
						continue
					} else if v.Key == "name" && v.Val == "SAMLResponse" {
						SamlResponse = token
						continue
					}
				}
			}
		case html.StartTagToken:
			if token.DataAtom == atom.P {
				for _, v := range token.Attr {
					if v.Key == "class" && v.Val == "fail-message" {
						log.Println("qtaphelper - User login failed")
						return "", errors.New("Bad username or password")
					}
				}
			}
		}
	}

	// Now we have to extract the attributes from the attribute slice
	var relayStateToken, samlResponseToken string
	for _, v := range relayState.Attr {
		if v.Key == "value" {
			relayStateToken = v.Val
			break
		}
	}

	for _, v := range SamlResponse.Attr {
		if v.Key == "value" {
			samlResponseToken = v.Val
			break
		}
	}

	// Now, we need to submit the Javascript form, aka submit the hidden values
	postValues = url.Values{}
	postValues.Add("RelayState", relayStateToken)
	postValues.Add("SAMLResponse", samlResponseToken)

	// Be good citizens and close
	resp.Body.Close()

	// Actually post it - this will redirect us to our target page
	resp, err = client.PostForm("https://my.queensu.ca/Shibboleth.sso/SAML2/POST", postValues)
	// This is our last request, so we can defer closing until the function exits
	defer resp.Body.Close()

	// We are not going to parse the DOM, since it wouldn't really help here. String manipulation instead.

	// Read the entire respones into a []byte
	bodyByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading page: ", err)
		return "", errors.New("Error reading page")
	}

	// Convert to a string so we can use strings package
	wholeBody := string(bodyByte)

	// Find the start index of the URL we desire
	urlStartIndex := strings.Index(wholeBody, "https://mytimetable.queensu.ca")
	// Find the bit towards the end, and advance the index appropriately (the length of the string we found)
	urlEndIndex := strings.Index(wholeBody, ".ics") + len(".ics")
	// Use slice syntax to access this portion of the URL
	urlString := wholeBody[urlStartIndex:urlEndIndex]

	// We're done! No error.
	return urlString, nil
}
