export interface Preview {
  id: string;
  source_url: string;
  final_url?: string;
  minio_key: string;
  content_type: string;
  download_url: string;
  created_at: string;
  expires_at?: string;
}
