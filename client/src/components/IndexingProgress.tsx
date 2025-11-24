import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Loader2, X, CheckCircle2 } from "lucide-react";
import { toast } from "sonner";
import { cancelJob, listJobs, listVideos, JobRecord, VideoRecord } from "@/lib/api";

interface EnrichedJob extends JobRecord {
  videoPath?: string;
}

export const IndexingProgress = () => {
  const [jobs, setJobs] = useState<EnrichedJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastRefetch, setLastRefetch] = useState<Date | null>(null);

  const fetchJobs = async () => {
    try {
      const [jobList, videos] = await Promise.all([listJobs(), listVideos()]);
      const videoMap = new Map<string, VideoRecord>();
      videos.forEach((v) => videoMap.set(v.id, v));
      const active = jobList.filter((job) => job.status !== "done" && job.status !== "failed");
      setJobs(
        active.map((job) => ({
          ...job,
          videoPath: videoMap.get(job.videoId)?.path,
        }))
      );
      setLastRefetch(new Date());
    } catch (error) {
      console.error("Failed to fetch jobs:", error);
      toast.error("Failed to load jobs");
    } finally {
      setLoading(false);
    }
  };

  const handleCancelJob = async (videoId: string) => {
    try {
      await cancelJob(videoId);
      toast.success("Job cancelled");
      fetchJobs();
    } catch (error) {
      toast.error("Failed to cancel job");
    }
  };

  useEffect(() => {
    fetchJobs();
    const interval = setInterval(fetchJobs, 3000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <Card className="p-4 gradient-card border-border/50">
        <div className="flex items-center gap-3">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="text-sm text-muted-foreground">Loading jobs...</span>
        </div>
      </Card>
    );
  }

  if (jobs.length === 0) {
    return (
      <Card className="p-6 gradient-card border-border/50 text-center">
        <CheckCircle2 className="h-12 w-12 text-success mx-auto mb-3 opacity-50" />
        <p className="text-sm text-muted-foreground">No active indexing jobs</p>
        {lastRefetch && (
          <p className="text-xs text-muted-foreground mt-1">
            Updated {lastRefetch.toLocaleTimeString()}
          </p>
        )}
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Indexing Progress</h3>
        <div className="flex items-center gap-2">
          <Badge variant="secondary">
            {jobs.length} active {jobs.length === 1 ? 'job' : 'jobs'}
          </Badge>
          {lastRefetch && (
            <span className="text-xs text-muted-foreground">
              Updated {lastRefetch.toLocaleTimeString()}
            </span>
          )}
        </div>
      </div>

      {jobs.map((job) => {
        const pct = Math.round(job.progress * 100);
        return (
          <Card key={job.id} className="p-4 gradient-card border-border/50">
            <div className="space-y-3">
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-3 flex-1 min-w-0">
                  <Loader2 className="h-5 w-5 animate-spin text-primary shrink-0" />
                  <div className="flex-1 min-w-0">
                    <h4 className="font-medium truncate">{job.videoPath || job.videoId}</h4>
                    <p className="text-xs text-muted-foreground">
                      {job.type} • {job.status}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  <Badge variant="secondary" className="font-mono">
                    {pct}%
                  </Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleCancelJob(job.videoId)}
                    className="hover:bg-destructive/10 hover:text-destructive"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <Progress value={pct} className="h-2" />

              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>Created {new Date(job.createdAt).toLocaleTimeString()}</span>
                <span className="font-mono">Updated {new Date(job.updatedAt).toLocaleTimeString()}</span>
              </div>
            </div>
          </Card>
        );
      })}
    </div>
  );
};
