import DownloadPageClient from "./download-page-client";

export default async function DownloadPage(props: PageProps<"/[uploadId]/[code]">) {
  const { uploadId, code } = await props.params;
  return <DownloadPageClient uploadId={uploadId} code={code} />;
}
