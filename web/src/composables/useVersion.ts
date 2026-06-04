import { ref, onMounted } from 'vue';

export interface VersionInfo {
  version: string;
  build_date: string;
  git_commit: string;
}

export function useVersion() {
  const version = ref('v0.5.1');
  const buildDate = ref('');
  const gitCommit = ref('');

  const fetchVersion = async () => {
    try {
      const res = await fetch('/api/v1/version');
      if (res.ok) {
        const data: VersionInfo = await res.json();
        version.value = 'v' + data.version;
        buildDate.value = data.build_date;
        gitCommit.value = data.git_commit;
      }
    } catch (e) {
      console.warn('Failed to fetch version:', e);
    }
  };

  onMounted(fetchVersion);

  return {
    version,
    buildDate,
    gitCommit,
  };
}
