import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { 
  Video, 
  Play, 
  Clock, 
  CheckCircle2,
  Loader2,
  RefreshCcw
} from "lucide-react";
import { toast } from "sonner";
import { addFolder, addVideo, listVideos, startExtract, VideoRecord } from "@/lib/api";
import { Label } from "@/components/ui/label";

interface VideoLibraryProps {
  onSelectVideo?: (video: VideoRecord) => void;
}

export const VideoLibrary = ({ onSelectVideo }: VideoLibraryProps) => {
  const [videos, setVideos] = useState<VideoRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [recursive, setRecursive] = useState(false);
  const [bulkAdding, setBulkAdding] = useState(false);

  const ensureElectron = () => {
    const api = window.electronAPI;
    if (!api) {
      toast.error("Electron APIs not available. Run via electron:dev.");
      return null;
    }
    return api;
  };

  const fetchVideos = async () => {
    try {
      setLoading(true);
      const data = await listVideos();
      setVideos(data);
    } catch (error) {
      console.error("Failed to fetch videos:", error);
      toast.error("Failed to load video library");
    } finally {
      setLoading(false);
    }
  };

  const chooseFolderAndScan = async () => {
    const api = ensureElectron();
    if (!api) return;
    const result = await api.chooseDirectory({ recursive });
    if (!result?.paths?.length) return;
    setBulkAdding(true);
    try {
      for (const folderPath of result.paths) {
        await addFolder(folderPath, recursive);
      }
      toast.success(`Added ${result.paths.length} folder(s) for scanning`);
      fetchVideos();
    } catch (error) {
      toast.error("Failed to add folder(s)");
    } finally {
      setBulkAdding(false);
    }
  };

  const chooseFileAndAdd = async () => {
    const api = ensureElectron();
    if (!api) return;
    const filePath = await api.chooseFile();
    if (!filePath) return;
    try {
      const res = await addVideo(filePath);
      if (res.status !== "already_exists") {
        await startExtract(res.videoId);
        toast.success("Video registered and extraction started");
      } else {
        toast.info("Video already exists");
      }
      fetchVideos();
    } catch (error) {
      toast.error("Failed to add video");
    }
  };

  const formatProgress = (video: VideoRecord) => {
    const total = video.totalFramesExpected || video.framesExtracted || 0;
    const indexed = video.framesUploaded ?? video.framesExtracted ?? 0;
    if (!total) return "pending";
    const pct = Math.min(100, Math.round((indexed / total) * 100));
    return `${pct}% indexed`;
  };

  useEffect(() => {
    fetchVideos();
  }, []);

  return (
    <div className="space-y-6">
      {/* Add Folder Section */}
      <div className="grid gap-3 md:grid-cols-2">
        <Card className="p-4 gradient-card border-border/50 space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-semibold">Add Folder</p>
              <p className="text-xs text-muted-foreground">Register a folder and start indexing videos.</p>
            </div>
            <Button
              variant="outline"
              onClick={chooseFolderAndScan}
              disabled={bulkAdding}
            >
              Choose Folder
            </Button>
          </div>
          <div className="flex items-center justify-between rounded-lg border border-border/60 px-3 py-2 bg-background/60">
            <label className="flex items-center gap-3 cursor-pointer select-none">
              <div
                className={`h-5 w-5 rounded-md border transition-colors ${
                  recursive ? "bg-primary border-primary" : "bg-background border-border"
                } relative flex items-center justify-center`}
              >
                {recursive && <CheckCircle2 className="h-4 w-4 text-primary-foreground" />}
                <input
                  id="recursive"
                  type="checkbox"
                  checked={recursive}
                  onChange={(e) => setRecursive(e.target.checked)}
                  className="absolute inset-0 opacity-0 cursor-pointer"
                />
              </div>
              <div className="flex flex-col leading-tight">
                <span className="text-sm font-medium">Recursive scan</span>
                <span className="text-xs text-muted-foreground">Include subfolders</span>
              </div>
            </label>
            <Badge variant="outline" className="text-xs">
              {recursive ? "Subfolders included" : "Top-level only"}
            </Badge>
          </div>
        </Card>

        <Card className="p-4 gradient-card border-border/50">
          <div className="flex flex-col gap-3">
            <Button
              variant="outline"
              onClick={chooseFileAndAdd}
            >
              Choose File
            </Button>
          </div>
        </Card>
      </div>

      {/* Videos Grid */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">Video Library</h3>
          <Badge variant="secondary">
            {videos.length} {videos.length === 1 ? 'video' : 'videos'}
          </Badge>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <div className="grid gap-3">
            {videos.map((video) => (
              <Card 
                key={video.id}
                className="p-4 gradient-card border-border/50 hover:border-primary/30 transition-smooth cursor-pointer group"
                onClick={() => onSelectVideo?.(video)}
              >
                <div className="flex items-start gap-4">
                  <div className="p-3 rounded-lg bg-primary/10 text-primary shrink-0">
                    <Video className="h-5 w-5" />
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-3 mb-2">
                      <div className="flex-1 min-w-0">
                        <h4 className="font-semibold text-foreground truncate group-hover:text-primary transition-smooth">
                          {video.path.split("/").pop() || video.id}
                        </h4>
                        <p className="text-xs text-muted-foreground truncate mt-1">
                          {video.path}
                        </p>
                      </div>

                      <Badge 
                        variant={video.indexStatus === "indexed" ? "default" : "secondary"}
                        className={video.indexStatus === "indexed" ? "bg-success/20 text-success border-success/30" : ""}
                      >
                        {video.indexStatus === "indexed" && <CheckCircle2 className="h-3 w-3 mr-1" />}
                        {video.indexStatus}
                      </Badge>
                    </div>

                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <div className="flex items-center gap-1.5">
                        <Clock className="h-3.5 w-3.5" />
                        {formatProgress(video)}
                      </div>
                      {video.lastIndexedAt && (
                        <div className="ml-auto">
                          Indexed {new Date(video.lastIndexedAt).toLocaleDateString()}
                        </div>
                      )}
                    </div>
                  </div>

                  {video.indexStatus === "indexed" && (
                    <Button 
                      variant="ghost" 
                      size="sm"
                      className="shrink-0 opacity-0 group-hover:opacity-100 transition-smooth"
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectVideo?.(video);
                        }}
                      >
                      <Play className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
