package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"chatgpt2api/handler"
	"chatgpt2api/internal/token"
)

func handleImageGenerations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model          string `json:"model"`
			Prompt         string `json:"prompt"`
			N              int    `json:"n"`
			Size           string `json:"size"`
			Quality        string `json:"quality"`
			Background     string `json:"background"`
			OutputFormat   string `json:"output_format"`
			ResponseFormat string `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Prompt == "" {
			writeError(w, http.StatusBadRequest, "prompt is required")
			return
		}
		if req.N < 1 {
			req.N = 1
		}
		if req.ResponseFormat == "" {
			req.ResponseFormat = "url"
		}

		tokenMgr := token.GetInstance()
		tokenMgr.ReloadIfStale()

		tokenStr := tokenMgr.GetToken(nil)
		if tokenStr == "" {
			writeError(w, http.StatusTooManyRequests, "no available tokens")
			return
		}

		client := handler.NewChatGPTClient(tokenStr, "")
		results, err := client.GenerateImage(r.Context(), req.Prompt, req.N, req.Size, req.Quality, req.Background)
		if err != nil {
			tokenMgr.RecordFail(tokenStr, err.Error())
			tokenMgr.Save()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tokenMgr.RecordSuccess(tokenStr)
		tokenMgr.Save()

		data := buildImageResponse(r, client, results, req.ResponseFormat)
		writeJSON(w, http.StatusOK, map[string]any{
			"created": time.Now().Unix(),
			"data":    data,
		})
	}
}

func handleImageEdits() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form (max 50MB)
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
			return
		}

		prompt := r.FormValue("prompt")
		if prompt == "" {
			writeError(w, http.StatusBadRequest, "prompt is required")
			return
		}
		responseFormat := r.FormValue("response_format")
		if responseFormat == "" {
			responseFormat = "url"
		}

		// Read image files — support both "image" and "image[]"
		var imageDataList [][]byte
		for _, key := range []string{"image", "image[]"} {
			files := r.MultipartForm.File[key]
			for _, fh := range files {
				f, err := fh.Open()
				if err != nil {
					writeError(w, http.StatusBadRequest, "cannot read image file: "+err.Error())
					return
				}
				data, err := io.ReadAll(f)
				f.Close()
				if err != nil {
					writeError(w, http.StatusBadRequest, "cannot read image file: "+err.Error())
					return
				}
				imageDataList = append(imageDataList, data)
			}
		}

		// Also support base64 image in form field "image_base64"
		if b64 := r.FormValue("image_base64"); b64 != "" {
			// Strip data URI prefix if present
			if idx := strings.Index(b64, ","); idx != -1 {
				b64 = b64[idx+1:]
			}
			decoded, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid base64 image: "+err.Error())
				return
			}
			imageDataList = append(imageDataList, decoded)
		}

		if len(imageDataList) == 0 {
			writeError(w, http.StatusBadRequest, "at least one image is required (use 'image' file field or 'image_base64')")
			return
		}

		// Read optional mask file
		var maskData []byte
		if maskFile, maskHeader, err := r.FormFile("mask"); err == nil {
			_ = maskHeader
			maskData, err = io.ReadAll(maskFile)
			maskFile.Close()
			if err != nil {
				writeError(w, http.StatusBadRequest, "cannot read mask file: "+err.Error())
				return
			}
		}

		tokenMgr := token.GetInstance()
		tokenMgr.ReloadIfStale()

		tokenStr := tokenMgr.GetToken(nil)
		if tokenStr == "" {
			writeError(w, http.StatusTooManyRequests, "no available tokens")
			return
		}

		client := handler.NewChatGPTClient(tokenStr, "")
		results, err := client.EditImageByUpload(r.Context(), prompt, imageDataList, maskData)
		if err != nil {
			tokenMgr.RecordFail(tokenStr, err.Error())
			tokenMgr.Save()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tokenMgr.RecordSuccess(tokenStr)
		tokenMgr.Save()

		data := buildImageResponse(r, client, results, responseFormat)
		writeJSON(w, http.StatusOK, map[string]any{
			"created": time.Now().Unix(),
			"data":    data,
		})
	}
}

// buildImageResponse converts ImageResults to the OpenAI-compatible response format.
// Only includes url/b64_json and revised_prompt — no internal ChatGPT fields.
func buildImageResponse(r *http.Request, client *handler.ChatGPTClient, results []handler.ImageResult, responseFormat string) []map[string]any {
	data := make([]map[string]any, 0, len(results))
	for _, img := range results {
		item := map[string]any{
			"revised_prompt": img.RevisedPrompt,
		}
		if responseFormat == "b64_json" {
			b64, err := client.DownloadAsBase64(r.Context(), img.URL)
			if err != nil {
				item["url"] = img.URL
			} else {
				item["b64_json"] = b64
			}
		} else {
			filename, err := downloadAndCache(client, img.URL)
			if err != nil {
				item["url"] = img.URL
			} else {
				item["url"] = gatewayImageURL(r, filename)
			}
		}
		data = append(data, item)
	}
	return data
}
