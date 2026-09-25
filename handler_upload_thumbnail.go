package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20 // 10MB
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}

	vid, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video likely doesn't exist", err)
		return
	}

	if userID != vid.UserID {
		respondWithError(w, http.StatusUnauthorized, "User does not own video", nil)
		return
	}

	mediaTypeSplit := strings.Split(mediaType, "/")
	fileExtension := mediaTypeSplit[len(mediaTypeSplit)-1]
	savePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoID, fileExtension))

	savedFile, err := os.Create(savePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating video file", err)
		return
	}

	io.Copy(savedFile, file)

	finalURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, savePath)

	vid.ThumbnailURL = &finalURL

	err = cfg.db.UpdateVideo(vid)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong updating vid data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vid)
}
