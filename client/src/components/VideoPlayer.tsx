import { useRef, useEffect } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Play, Pause, Volume2, VolumeX, Maximize, X } from "lucide-react";
import { useState } from "react";

interface VideoPlayerProps {
  videoUrl?: string;
  startTime?: number;
  filename?: string;
  onClose?: () => void;
}

export const VideoPlayer = ({ 
  videoUrl = "", 
  startTime = 0,
  filename = "video.mp4",
  onClose,
}: VideoPlayerProps) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const lastUrl = useRef<string>("");
  const [playing, setPlaying] = useState(false);
  const [muted, setMuted] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const sameSource = lastUrl.current === videoUrl && videoUrl !== "";
    setPlaying(false);
    setCurrentTime(startTime || 0);

    const handlePlay = () => setPlaying(true);
    const handlePause = () => setPlaying(false);
    video.addEventListener("play", handlePlay);
    video.addEventListener("pause", handlePause);

    const setTime = () => {
      try {
        video.currentTime = startTime || 0;
      } catch (_) {}
    };

    const handleMetadata = () => {
      setDuration(video.duration || 0);
      setTime();
      video.play().catch(() => {});
    };

    if (sameSource) {
      setTime();
      video.play().catch(() => {});
    } else {
      if (!video.paused) {
        video.pause();
      }
      video.addEventListener("loadedmetadata", handleMetadata);
      video.load();
      lastUrl.current = videoUrl;
    }

    return () => {
      video.removeEventListener("play", handlePlay);
      video.removeEventListener("pause", handlePause);
      video.removeEventListener("loadedmetadata", handleMetadata);
    };
  }, [videoUrl, startTime]);

  const togglePlay = () => {
    if (videoRef.current) {
      if (playing) {
        videoRef.current.pause();
        setPlaying(false);
      } else {
        videoRef.current.play();
        setPlaying(true);
      }
    }
  };

  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !muted;
      setMuted(!muted);
    }
  };

  const toggleFullscreen = () => {
    if (videoRef.current) {
      if (document.fullscreenElement) {
        document.exitFullscreen();
      } else {
        videoRef.current.requestFullscreen();
      }
    }
  };

  const handleTimeUpdate = () => {
    if (videoRef.current) {
      setCurrentTime(videoRef.current.currentTime);
    }
  };

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const time = parseFloat(e.target.value);
    if (videoRef.current) {
      videoRef.current.currentTime = time;
      setCurrentTime(time);
    }
  };

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const hasSource = Boolean(videoUrl);

  return (
    <Card className="overflow-hidden gradient-card border-border/50">
      <div className="relative group">
        {onClose && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            className="absolute top-2 right-2 z-10 text-white bg-black/40 hover:bg-black/60"
          >
            <X className="h-4 w-4" />
          </Button>
        )}
        {hasSource ? (
          <video
            ref={videoRef}
            src={videoUrl}
            className="w-full aspect-video bg-black"
            onTimeUpdate={handleTimeUpdate}
            onClick={togglePlay}
            autoPlay={startTime > 0}
          />
        ) : (
          <div className="w-full aspect-video bg-muted flex items-center justify-center text-muted-foreground text-sm">
            No video source available
          </div>
        )}

        {/* Video Controls Overlay */}
        {hasSource && (
          <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/90 to-transparent p-4 opacity-0 group-hover:opacity-100 transition-smooth">
            {/* Progress Bar */}
            <input
              type="range"
              min="0"
              max={duration || 0}
              value={currentTime}
              onChange={handleSeek}
              className="w-full h-1 mb-3 bg-white/20 rounded-lg appearance-none cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-primary"
            />

            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={togglePlay}
                  className="text-white hover:bg-white/20"
                >
                  {playing ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
                </Button>

                <Button
                  variant="ghost"
                  size="sm"
                  onClick={toggleMute}
                  className="text-white hover:bg-white/20"
                >
                  {muted ? <VolumeX className="h-5 w-5" /> : <Volume2 className="h-5 w-5" />}
                </Button>

                <span className="text-white text-sm font-mono">
                  {formatTime(currentTime)} / {formatTime(duration)}
                </span>
              </div>

              <div className="flex items-center gap-2">
                <span className="text-white text-xs truncate max-w-[200px]">
                  {filename}
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={toggleFullscreen}
                  className="text-white hover:bg-white/20"
                >
                  <Maximize className="h-5 w-5" />
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </Card>
  );
};
