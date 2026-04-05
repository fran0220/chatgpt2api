package api

import (
	"encoding/json"
	"net/http"
	"time"

	"chatgpt2api/handler"
	"chatgpt2api/internal/token"
)

func handleImageGenerations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Prompt         string `json:"prompt"`
			N              int    `json:"n"`
			Size           string `json:"size"`
			Quality        string `json:"quality"`
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
		results, err := client.GenerateImage(r.Context(), req.Prompt, req.N, req.Size, req.Quality)
		if err != nil {
			tokenMgr.RecordFail(tokenStr, err.Error())
			tokenMgr.Save()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tokenMgr.RecordSuccess(tokenStr)
		tokenMgr.Save()

		data := make([]map[string]any, 0, len(results))
		for _, img := range results {
			item := map[string]any{"revised_prompt": img.RevisedPrompt}
			if req.ResponseFormat == "b64_json" {
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

		writeJSON(w, http.StatusOK, map[string]any{
			"created": time.Now().Unix(),
			"data":    data,
		})
	}
}

func handleImageEdits() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Prompt         string `json:"prompt"`
			ImageFileID    string `json:"image_file_id"`
			GenID          string `json:"gen_id"`
			ConversationID string `json:"conversation_id"`
			ParentMsgID    string `json:"parent_message_id"`
			MaskFileID     string `json:"mask_file_id"`
			N              int    `json:"n"`
			ResponseFormat string `json:"response_format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Prompt == "" || req.ImageFileID == "" || req.GenID == "" {
			writeError(w, http.StatusBadRequest, "prompt, image_file_id, and gen_id are required")
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
		results, err := client.EditImage(r.Context(), handler.EditRequest{
			Prompt:         req.Prompt,
			ImageFileID:    req.ImageFileID,
			GenID:          req.GenID,
			ConversationID: req.ConversationID,
			ParentMsgID:    req.ParentMsgID,
			MaskFileID:     req.MaskFileID,
		})
		if err != nil {
			tokenMgr.RecordFail(tokenStr, err.Error())
			tokenMgr.Save()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tokenMgr.RecordSuccess(tokenStr)
		tokenMgr.Save()

		data := make([]map[string]any, 0, len(results))
		for _, img := range results {
			item := map[string]any{"revised_prompt": img.RevisedPrompt}
			if req.ResponseFormat == "b64_json" {
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

		writeJSON(w, http.StatusOK, map[string]any{
			"created": time.Now().Unix(),
			"data":    data,
		})
	}
}
