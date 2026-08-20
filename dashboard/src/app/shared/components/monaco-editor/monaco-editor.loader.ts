import {Injectable} from '@angular/core';

export type MonacoApi = typeof import('./monaco-editor-api');

interface MonacoEnvironment {
  getWorker(moduleId: string, label: string): Worker;
}

type MonacoGlobal = typeof globalThis & {
  MonacoEnvironment?: Partial<MonacoEnvironment>;
};

@Injectable({providedIn: 'root'})
export class MonacoEditorLoader {
  private loadPromise?: Promise<MonacoApi>;

  load(): Promise<MonacoApi> {
    if (!this.loadPromise) {
      this.configureWorkers();
      this.loadPromise = import('./monaco-editor-api');
    }
    return this.loadPromise;
  }

  private configureWorkers(): void {
    const monacoGlobal = globalThis as MonacoGlobal;
    monacoGlobal.MonacoEnvironment = {
      ...monacoGlobal.MonacoEnvironment,
      getWorker: (_moduleId: string, label: string) => {
        if (label === 'javascript' || label === 'typescript') {
          return new Worker(new URL('./typescript.worker', import.meta.url), {
            name: label,
            type: 'module',
          });
        }
        return new Worker(new URL('./editor.worker', import.meta.url), {
          name: label,
          type: 'module',
        });
      },
    };
  }
}
