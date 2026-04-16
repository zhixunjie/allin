  ffmpeg -y -ss 00:01:27 -to 00:02:16 -i docs/input.mp4 \
    -vf "fps=20,scale=1280:-1:flags=lanczos" \
    -vcodec libwebp -lossless 0 -quality 80 -preset picture -loop 0 \
    docs/output.webp