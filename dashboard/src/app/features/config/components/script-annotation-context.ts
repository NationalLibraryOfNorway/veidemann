import {Annotation, ConfigRef} from '../../../shared/models';

export const SCRIPT_ANNOTATION_DRAG_TYPE = 'application/x-veidemann-script-annotation';

export interface ScriptAnnotationContext {
  jobRef: ConfigRef;
  jobName: string;
  annotations: Annotation[];
  unavailable: boolean;
}
