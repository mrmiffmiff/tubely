package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
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

	vid, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video likely doesn't exist", err)
		return
	}

	if userID != vid.UserID {
		respondWithError(w, http.StatusUnauthorized, "User does not own video", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid video file type", nil)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temp file on server", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem copying file contents", err)
		return
	}

	ratio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem getting video aspect ratio", err)
		return
	}

	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem resetting tmp file pointer to start", err)
		return
	}

	othFilePath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error pre-processing video", err)
		return
	}
	othFile, err := os.Open(othFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error opening preprocessed video", err)
		return
	}
	defer os.Remove(othFile.Name())
	defer othFile.Close()

	randSlice := make([]byte, 32)
	_, err = rand.Read(randSlice)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Problem creating byte slice", err)
		return
	}
	randID := base64.RawURLEncoding.EncodeToString(randSlice)

	assetKey := getAssetPath(randID, mediaType)
	var prefix string
	switch ratio {
	case "16:9":
		prefix = "landscape"
	case "9:16":
		prefix = "portrait"
	default:
		prefix = "other"
	}
	fullKey := fmt.Sprintf("%s/%s", prefix, assetKey)

	_, err = cfg.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fullKey,
		Body:        othFile,
		ContentType: &mediaType,
	})

	finalURL := fmt.Sprintf("https://%s/%s", cfg.s3CfDistribution, fullKey)
	vid.VideoURL = &finalURL

	err = cfg.db.UpdateVideo(vid)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong updating vid data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vid)
}
