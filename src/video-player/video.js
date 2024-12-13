import React, { useRef, useEffect } from 'react';
import Hls from 'hls.js';

const VideoPlayer = ({ src }) => {
    const videoRef = useRef(null);

    useEffect(() => {
        if (Hls.isSupported()) {
            const hls = new Hls();
            hls.loadSource(src);
            hls.attachMedia(videoRef.current);
            hls.on(Hls.Events.ERROR, (event, data) => {
                console.error('HLS.js error:', data);
            });
        } else if (videoRef.current.canPlayType('application/vnd.apple.mpegurl')) {
            // Fallback for Safari
            videoRef.current.src = src;
        }
    }, [src]);

    return (
        <div>
            <video
                ref={videoRef}
                controls
                autoPlay
                muted
                style={{ width: '100%', height: 'auto' }}
            />
        </div>
    );
};

export default VideoPlayer;
