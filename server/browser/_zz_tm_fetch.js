// ==UserScript==
// @name         SSE (Server-Sent Events) Listener
// @namespace    https://www.tampermonkey.net/
// @version      1.0
// @description  Listens to fetch requests and logs data from Server-Sent Events (SSE) streams.
// @author       aichat-proxy
// @match        *://*/*
// @grant        unsafeWindow
// @run-at       document-start
// ==/UserScript==

(function () {
  "use strict"

  // Store the original fetch function to be called later.
  const originalFetch = unsafeWindow.fetch

  async function logSseEvents(stream) {
    const reader = stream.getReader()
    const decoder = new TextDecoder()
    let buffer = ""

    console.info("[aichat-proxy-sse-new]")

    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) {
          console.info("[aichat-proxy-sse-closed]")
          break
        }

        // Append the new chunk of data to our buffer.
        buffer += decoder.decode(value, { stream: true })

        // Process all complete messages in the buffer.
        // SSE messages are separated by double newlines ('\n\n').
        let boundaryIndex
        while ((boundaryIndex = buffer.indexOf("\n\n")) >= 0) {
          // Extract one complete message.
          const message = buffer.slice(0, boundaryIndex)
          // Remove the processed message from the buffer.
          buffer = buffer.slice(boundaryIndex + 2)

          if (message) {
            // Extract only the data from lines starting with "data:".
            const dataLines = message.split("\n")
              .filter(line => line.startsWith("data:"))
              .map(line => line.substring(5).trim()) // 5 is the length of "data:"

            if (dataLines.length > 0) {
              // Join multi-line data fields and log the final event data.
              const eventData = dataLines.join("\n")
              console.info("[aichat-proxy-sse-data]", eventData)
            }
          }
        }
      }
    } catch (error) {
      console.info("[aichat-proxy-sse-error]", JSON.stringify(error))
    }
  }

  // Replace the global fetch function with our custom version.
  unsafeWindow.fetch = function (...args) {
    // Call the original fetch and handle its response in a promise.
    return originalFetch.apply(this, args).then(response => {

      const contentType = response.headers.get("Content-Type")

      // Check if the response is an SSE stream.
      if (contentType && contentType.includes("text/event-stream")) {
        // Tee the stream to allow both the page and our script to read it.
        const [streamForPage, streamForLogger] = response.body.tee()

        // Start logging the events from our copy of the stream in the background.
        logSseEvents(streamForLogger).then()

        // Return a new response with the other stream to the original caller.
        // This ensures the page works as expected.
        return new Response(streamForPage, {
          headers: response.headers, status: response.status, statusText: response.statusText,
        })
      }

      // For all non-SSE requests, return the original response without any changes.
      return response
    }).catch(error => {
      // Log any errors during the fetch process and re-throw them.
      console.info("[aichat-proxy-sse-error]", JSON.stringify(error))
      throw error
    })
  }
})()
