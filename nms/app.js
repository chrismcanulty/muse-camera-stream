const NodeMediaServer = require('node-media-server');
const fs = require('fs');
const moment = require('moment');
const ffmpeg = require('fluent-ffmpeg');

// NodeMediaServer configuration
const config = {
  rtmp: {
    port: 1935,
    chunk_size: 60000,
    gop_cache: true,
    ping: 60,
    ping_timeout: 30,
  },
  http: {
    port: 8000,
    allow_origin: '*',
    mediaroot: './media',
    webroot: './www',
    api: true,
    hls: true,
    hlsflags: '[hls_time=2:hls_list_size=3:hls_flags=delete_segments]',
    headers: {
      'Content-Type': 'application/vnd.apple.mpegurl',
      'Access-Control-Allow-Origin': '*',
    },
  },
  trans: {
    ffmpeg: '/opt/homebrew/bin/ffmpeg',
    tasks: [
      {
        app: 'live',
        name: 'camera1',
        input: 'rtsp://Musecamera:Muse2024!@174.6.124.37:554/stream1',
        hls: true,
        hlsFlags: '[hls_time=2:hls_list_size=3:hls_flags=delete_segments]',
        ffmpegOptions: '-fflags +igndts -use_wallclock_as_timestamps 1',
        output: './media/live/camera1',
      },
    ],
  },
  logType: 3,
};

// Ensure output directory exists
const outputDir = './media/live/camera1/';
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

// Flag to track if FFmpeg is already running
let ffmpegRunning = false;

// Save stream function with Fluent-FFmpeg
function saveStream() {
  // Don't start a new FFmpeg process if one is already running
  if (ffmpegRunning) {
    console.log("FFmpeg process is already running.");
    return;
  }

  const timestamp = moment().format('YYYY-MM-DD_HH-mm-ss');
  const outputFilename = `${outputDir}stream.m3u8`;

  ffmpeg('rtsp://Musecamera:Muse2024!@174.6.124.37:554/stream1')
    .inputFormat('rtsp')
    .output(outputFilename)
    .format('hls')
    .outputOptions([
      '-hls_time 2',
      '-hls_list_size 3',
      '-hls_flags delete_segments',
      '-movflags frag_keyframe+empty_moov',
    ])
    .on('start', (cmd) => {
      console.log(`FFmpeg started: ${cmd}`);
      ffmpegRunning = true;
    })
    .on('end', () => {
      console.log(`Stream saved to ${outputFilename}`);
      ffmpegRunning = false;
    })
    .on('error', (err, stdout, stderr) => {
      console.error('FFmpeg error:', err.message);
      console.error('FFmpeg stdout:', stdout);
      console.error('FFmpeg stderr:', stderr);
      ffmpegRunning = false;
    })
    .run();

}

// Initialize NodeMediaServer
const nms = new NodeMediaServer(config);

// Handle events
nms.on('postPublish', (_, streamPath, _params) => {
  console.log(`Stream started: ${streamPath}`);
  saveStream(); // Start saving the stream upon publication
});

// Start the server
nms.run();

// Optional: Periodically re-save the stream (use with caution to avoid redundant saves)
setInterval(() => {
  if (!ffmpegRunning) {
    saveStream();
  }
}, 30000);
