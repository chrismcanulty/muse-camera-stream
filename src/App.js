import './App.css';
import VideoPlayer from './video-player/video';

function App() {
  return (
    <div className="App-header">
      <p>Proof of concept for camera streaming</p>
      <VideoPlayer src="http://localhost:8000/live/camera1/stream.m3u8" />
    </div>

  );
}

export default App;
