//go:build gui_ffmpeg_cgo

package ffmpeg

/*
#cgo pkg-config: libavformat libavcodec libavutil libswresample
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/channel_layout.h>
#include <libavutil/error.h>
#include <libavutil/frame.h>
#include <libavutil/mem.h>
#include <libavutil/opt.h>
#include <libavutil/samplefmt.h>
#include <libswresample/swresample.h>

static void audiobook_set_error(char *errbuf, int errbuf_size, const char *message) {
	if (errbuf != NULL && errbuf_size > 0) {
		snprintf(errbuf, errbuf_size, "%s", message);
	}
}

static void audiobook_set_av_error(char *errbuf, int errbuf_size, const char *prefix, int err) {
	char av_error[AV_ERROR_MAX_STRING_SIZE] = {0};
	av_strerror(err, av_error, sizeof(av_error));
	if (errbuf != NULL && errbuf_size > 0) {
		snprintf(errbuf, errbuf_size, "%s: %s", prefix, av_error);
	}
}

static int audiobook_append_pcm(uint8_t **out, int *out_len, const uint8_t *data, int data_len, char *errbuf, int errbuf_size) {
	uint8_t *grown = av_realloc(*out, *out_len + data_len);
	if (grown == NULL) {
		audiobook_set_error(errbuf, errbuf_size, "could not grow pcm buffer");
		return AVERROR(ENOMEM);
	}
	memcpy(grown + *out_len, data, data_len);
	*out = grown;
	*out_len += data_len;
	return 0;
}

static int audiobook_decode_file_to_pcm(const char *in_path, uint8_t **out, int *out_len, int target_rate, char *errbuf, int errbuf_size) {
	AVFormatContext *fmt = NULL;
	AVCodecContext *dec = NULL;
	AVPacket *pkt = NULL;
	AVFrame *frame = NULL;
	SwrContext *swr = NULL;
	int audio_stream = -1;
	int ret = 0;
	AVChannelLayout mono;
	av_channel_layout_default(&mono, 1);

	*out = NULL;
	*out_len = 0;
	av_log_set_level(AV_LOG_ERROR);

	ret = avformat_open_input(&fmt, in_path, NULL, NULL);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "open input", ret); goto cleanup; }
	ret = avformat_find_stream_info(fmt, NULL);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "read stream info", ret); goto cleanup; }
	audio_stream = av_find_best_stream(fmt, AVMEDIA_TYPE_AUDIO, -1, -1, NULL, 0);
	if (audio_stream < 0) { ret = audio_stream; audiobook_set_av_error(errbuf, errbuf_size, "find audio stream", ret); goto cleanup; }

	const AVCodec *decoder = avcodec_find_decoder(fmt->streams[audio_stream]->codecpar->codec_id);
	if (decoder == NULL) { ret = AVERROR_DECODER_NOT_FOUND; audiobook_set_error(errbuf, errbuf_size, "decoder not found"); goto cleanup; }
	dec = avcodec_alloc_context3(decoder);
	if (dec == NULL) { ret = AVERROR(ENOMEM); audiobook_set_error(errbuf, errbuf_size, "allocate decoder"); goto cleanup; }
	ret = avcodec_parameters_to_context(dec, fmt->streams[audio_stream]->codecpar);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "copy decoder params", ret); goto cleanup; }
	if (dec->ch_layout.nb_channels <= 0) {
		av_channel_layout_default(&dec->ch_layout, 1);
	}
	ret = avcodec_open2(dec, decoder, NULL);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "open decoder", ret); goto cleanup; }

	ret = swr_alloc_set_opts2(&swr, &mono, AV_SAMPLE_FMT_S16, target_rate, &dec->ch_layout, dec->sample_fmt, dec->sample_rate, 0, NULL);
	if (ret < 0 || swr == NULL) { audiobook_set_av_error(errbuf, errbuf_size, "allocate resampler", ret); goto cleanup; }
	ret = swr_init(swr);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "init resampler", ret); goto cleanup; }

	pkt = av_packet_alloc();
	frame = av_frame_alloc();
	if (pkt == NULL || frame == NULL) { ret = AVERROR(ENOMEM); audiobook_set_error(errbuf, errbuf_size, "allocate packet/frame"); goto cleanup; }

	while ((ret = av_read_frame(fmt, pkt)) >= 0) {
		if (pkt->stream_index != audio_stream) {
			av_packet_unref(pkt);
			continue;
		}
		ret = avcodec_send_packet(dec, pkt);
		av_packet_unref(pkt);
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "send packet", ret); goto cleanup; }
		while ((ret = avcodec_receive_frame(dec, frame)) >= 0) {
			int out_samples = (int)av_rescale_rnd(swr_get_delay(swr, dec->sample_rate) + frame->nb_samples, target_rate, dec->sample_rate, AV_ROUND_UP);
			uint8_t **converted = NULL;
			ret = av_samples_alloc_array_and_samples(&converted, NULL, 1, out_samples, AV_SAMPLE_FMT_S16, 0);
			if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "allocate converted samples", ret); goto cleanup; }
			ret = swr_convert(swr, converted, out_samples, (const uint8_t **)frame->extended_data, frame->nb_samples);
			if (ret < 0) {
				av_freep(&converted[0]);
				av_freep(&converted);
				audiobook_set_av_error(errbuf, errbuf_size, "resample frame", ret);
				goto cleanup;
			}
			int bytes = ret * 2;
			ret = audiobook_append_pcm(out, out_len, converted[0], bytes, errbuf, errbuf_size);
			av_freep(&converted[0]);
			av_freep(&converted);
			if (ret < 0) { goto cleanup; }
			av_frame_unref(frame);
		}
		if (ret != AVERROR(EAGAIN) && ret != AVERROR_EOF) { audiobook_set_av_error(errbuf, errbuf_size, "receive frame", ret); goto cleanup; }
	}
	if (ret == AVERROR_EOF) { ret = 0; }
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "read packet", ret); goto cleanup; }

	ret = avcodec_send_packet(dec, NULL);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "flush decoder", ret); goto cleanup; }
	while ((ret = avcodec_receive_frame(dec, frame)) >= 0) {
		int out_samples = (int)av_rescale_rnd(swr_get_delay(swr, dec->sample_rate) + frame->nb_samples, target_rate, dec->sample_rate, AV_ROUND_UP);
		uint8_t **converted = NULL;
		ret = av_samples_alloc_array_and_samples(&converted, NULL, 1, out_samples, AV_SAMPLE_FMT_S16, 0);
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "allocate flushed samples", ret); goto cleanup; }
		ret = swr_convert(swr, converted, out_samples, (const uint8_t **)frame->extended_data, frame->nb_samples);
		if (ret < 0) {
			av_freep(&converted[0]);
			av_freep(&converted);
			audiobook_set_av_error(errbuf, errbuf_size, "resample flushed frame", ret);
			goto cleanup;
		}
		int bytes = ret * 2;
		ret = audiobook_append_pcm(out, out_len, converted[0], bytes, errbuf, errbuf_size);
		av_freep(&converted[0]);
		av_freep(&converted);
		if (ret < 0) { goto cleanup; }
		av_frame_unref(frame);
	}
	if (ret == AVERROR_EOF || ret == AVERROR(EAGAIN)) { ret = 0; }

cleanup:
	if (ret < 0 && *out != NULL) {
		av_freep(out);
		*out_len = 0;
	}
	swr_free(&swr);
	av_frame_free(&frame);
	av_packet_free(&pkt);
	avcodec_free_context(&dec);
	avformat_close_input(&fmt);
	av_channel_layout_uninit(&mono);
	return ret;
}

static enum AVSampleFormat audiobook_pick_sample_fmt(const AVCodec *codec) {
	if (codec->sample_fmts == NULL) {
		return AV_SAMPLE_FMT_S16P;
	}
	const enum AVSampleFormat *p = codec->sample_fmts;
	while (*p != AV_SAMPLE_FMT_NONE) {
		if (*p == AV_SAMPLE_FMT_S16P) {
			return *p;
		}
		p++;
	}
	return codec->sample_fmts[0];
}

static int audiobook_encode_pcm_to_mp3(const char *out_path, const uint8_t *pcm, int pcm_len, int sample_rate, int bitrate_kbps, char *errbuf, int errbuf_size) {
	AVFormatContext *fmt = NULL;
	AVCodecContext *enc = NULL;
	AVStream *stream = NULL;
	SwrContext *swr = NULL;
	AVFrame *frame = NULL;
	AVPacket *pkt = NULL;
	int ret = 0;
	int64_t pts = 0;
	AVChannelLayout mono;
	av_channel_layout_default(&mono, 1);

	av_log_set_level(AV_LOG_ERROR);
	const AVCodec *codec = avcodec_find_encoder_by_name("libmp3lame");
	if (codec == NULL) {
		codec = avcodec_find_encoder(AV_CODEC_ID_MP3);
	}
	if (codec == NULL) { ret = AVERROR_ENCODER_NOT_FOUND; audiobook_set_error(errbuf, errbuf_size, "mp3 encoder not found"); goto cleanup; }

	ret = avformat_alloc_output_context2(&fmt, NULL, "mp3", out_path);
	if (ret < 0 || fmt == NULL) { audiobook_set_av_error(errbuf, errbuf_size, "allocate output", ret); goto cleanup; }
	stream = avformat_new_stream(fmt, NULL);
	if (stream == NULL) { ret = AVERROR(ENOMEM); audiobook_set_error(errbuf, errbuf_size, "allocate output stream"); goto cleanup; }
	enc = avcodec_alloc_context3(codec);
	if (enc == NULL) { ret = AVERROR(ENOMEM); audiobook_set_error(errbuf, errbuf_size, "allocate encoder"); goto cleanup; }
	enc->bit_rate = bitrate_kbps * 1000;
	enc->sample_rate = sample_rate;
	enc->sample_fmt = audiobook_pick_sample_fmt(codec);
	ret = av_channel_layout_copy(&enc->ch_layout, &mono);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "copy channel layout", ret); goto cleanup; }
	stream->time_base = (AVRational){1, sample_rate};
	enc->time_base = stream->time_base;
	if (fmt->oformat->flags & AVFMT_GLOBALHEADER) {
		enc->flags |= AV_CODEC_FLAG_GLOBAL_HEADER;
	}
	ret = avcodec_open2(enc, codec, NULL);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "open encoder", ret); goto cleanup; }
	ret = avcodec_parameters_from_context(stream->codecpar, enc);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "copy encoder params", ret); goto cleanup; }
	ret = avio_open(&fmt->pb, out_path, AVIO_FLAG_WRITE);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "open output", ret); goto cleanup; }
	ret = avformat_write_header(fmt, NULL);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "write header", ret); goto cleanup; }

	ret = swr_alloc_set_opts2(&swr, &enc->ch_layout, enc->sample_fmt, enc->sample_rate, &mono, AV_SAMPLE_FMT_S16, sample_rate, 0, NULL);
	if (ret < 0 || swr == NULL) { audiobook_set_av_error(errbuf, errbuf_size, "allocate encoder resampler", ret); goto cleanup; }
	ret = swr_init(swr);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "init encoder resampler", ret); goto cleanup; }

	int frame_samples = enc->frame_size > 0 ? enc->frame_size : 1152;
	int total_samples = pcm_len / 2;
	int offset = 0;
	while (offset < total_samples) {
		int samples = frame_samples;
		if (samples > total_samples - offset) {
			samples = total_samples - offset;
		}
		frame = av_frame_alloc();
		if (frame == NULL) { ret = AVERROR(ENOMEM); audiobook_set_error(errbuf, errbuf_size, "allocate encode frame"); goto cleanup; }
		frame->format = enc->sample_fmt;
		frame->sample_rate = enc->sample_rate;
		frame->nb_samples = samples;
		ret = av_channel_layout_copy(&frame->ch_layout, &enc->ch_layout);
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "copy frame layout", ret); goto cleanup; }
		ret = av_frame_get_buffer(frame, 0);
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "allocate frame buffer", ret); goto cleanup; }
		const uint8_t *src = pcm + offset * 2;
		ret = swr_convert(swr, frame->extended_data, samples, &src, samples);
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "convert pcm for encoder", ret); goto cleanup; }
		frame->nb_samples = ret;
		frame->pts = pts;
		pts += frame->nb_samples;
		ret = avcodec_send_frame(enc, frame);
		av_frame_free(&frame);
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "send encode frame", ret); goto cleanup; }
		while (1) {
			pkt = av_packet_alloc();
			if (pkt == NULL) { ret = AVERROR(ENOMEM); audiobook_set_error(errbuf, errbuf_size, "allocate packet"); goto cleanup; }
			ret = avcodec_receive_packet(enc, pkt);
			if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
				av_packet_free(&pkt);
				ret = 0;
				break;
			}
			if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "receive encoded packet", ret); goto cleanup; }
			av_packet_rescale_ts(pkt, enc->time_base, stream->time_base);
			pkt->stream_index = stream->index;
			ret = av_interleaved_write_frame(fmt, pkt);
			av_packet_free(&pkt);
			if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "write encoded packet", ret); goto cleanup; }
		}
		offset += samples;
	}
	ret = avcodec_send_frame(enc, NULL);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "flush encoder", ret); goto cleanup; }
	while (1) {
		pkt = av_packet_alloc();
		if (pkt == NULL) { ret = AVERROR(ENOMEM); audiobook_set_error(errbuf, errbuf_size, "allocate flush packet"); goto cleanup; }
		ret = avcodec_receive_packet(enc, pkt);
		if (ret == AVERROR_EOF || ret == AVERROR(EAGAIN)) {
			av_packet_free(&pkt);
			ret = 0;
			break;
		}
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "receive flush packet", ret); goto cleanup; }
		av_packet_rescale_ts(pkt, enc->time_base, stream->time_base);
		pkt->stream_index = stream->index;
		ret = av_interleaved_write_frame(fmt, pkt);
		av_packet_free(&pkt);
		if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "write flush packet", ret); goto cleanup; }
	}
	ret = av_write_trailer(fmt);
	if (ret < 0) { audiobook_set_av_error(errbuf, errbuf_size, "write trailer", ret); goto cleanup; }

cleanup:
	av_packet_free(&pkt);
	av_frame_free(&frame);
	swr_free(&swr);
	avcodec_free_context(&enc);
	if (fmt != NULL) {
		if (fmt->pb != NULL) {
			avio_closep(&fmt->pb);
		}
		avformat_free_context(fmt);
	}
	av_channel_layout_uninit(&mono);
	return ret;
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"
)

func decodeToPCM(e Encoder, input []byte) ([]byte, int, error) {
	tmp, err := os.CreateTemp("", "audiobook-studio-*.audio")
	if err != nil {
		return nil, 0, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(input); err != nil {
		_ = tmp.Close()
		return nil, 0, err
	}
	if err := tmp.Close(); err != nil {
		return nil, 0, err
	}

	cPath := C.CString(tmpName)
	defer C.free(unsafe.Pointer(cPath))
	errbuf := (*C.char)(C.calloc(1, 1024))
	defer C.free(unsafe.Pointer(errbuf))
	var out *C.uint8_t
	var outLen C.int
	if ret := C.audiobook_decode_file_to_pcm(cPath, &out, &outLen, C.int(sampleRate), errbuf, 1024); ret < 0 {
		return nil, 0, fmt.Errorf("ffmpeg cgo decode failed: %s", C.GoString(errbuf))
	}
	defer C.av_free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), outLen), sampleRate, nil
}

func mergeToMP3(e Encoder, segments [][]byte, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	var pcm bytes.Buffer
	for _, segment := range segments {
		pcm.Write(segment)
	}
	data := pcm.Bytes()
	if len(data) == 0 {
		return fmt.Errorf("no pcm data to encode")
	}
	cPath := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cPath))
	errbuf := (*C.char)(C.calloc(1, 1024))
	defer C.free(unsafe.Pointer(errbuf))
	if ret := C.audiobook_encode_pcm_to_mp3(cPath, (*C.uint8_t)(unsafe.Pointer(&data[0])), C.int(len(data)), C.int(sampleRate), C.int(e.OutputBitrateKB), errbuf, 1024); ret < 0 {
		return fmt.Errorf("ffmpeg cgo encode failed: %s", C.GoString(errbuf))
	}
	return nil
}
