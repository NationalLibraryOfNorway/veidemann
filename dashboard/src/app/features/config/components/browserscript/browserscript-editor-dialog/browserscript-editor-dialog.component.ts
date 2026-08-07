import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {toSignal} from '@angular/core/rxjs-interop';
import {EditorComponent} from 'ngx-monaco-editor-v2';

export interface BrowserScriptEditorDialogData {
  name: string;
  script: string;
  readOnly: boolean;
  theme: string;
}

export interface BrowserScriptEditorDialogResult {
  script: string;
}

@Component({
  selector: 'app-browserscript-editor-dialog',
  templateUrl: './browserscript-editor-dialog.component.html',
  styleUrls: ['./browserscript-editor-dialog.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    EditorComponent,
    MatButtonModule,
    MatDialogModule,
    ReactiveFormsModule,
  ],
  standalone: true,
})
export class BrowserScriptEditorDialogComponent {
  private readonly dialogRef = inject<MatDialogRef<BrowserScriptEditorDialogComponent, BrowserScriptEditorDialogResult>>(
    MatDialogRef
  );
  readonly data = inject<BrowserScriptEditorDialogData>(MAT_DIALOG_DATA);

  readonly scriptControl = new FormControl(this.data.script, {nonNullable: true});
  private readonly scriptValue = toSignal(this.scriptControl.valueChanges, {initialValue: this.data.script});
  readonly editorOptions = {
    theme: this.data.theme,
    language: 'javascript',
    roundedSelection: true,
    automaticLayout: true,
    readOnly: this.data.readOnly,
  };

  get canApply(): boolean {
    return !this.data.readOnly
      && this.scriptValue() !== this.data.script;
  }

  get canRevert(): boolean {
    return !this.data.readOnly
      && this.scriptValue() !== this.data.script;
  }

  onApply(): void {
    if (!this.canApply) {
      return;
    }

    this.dialogRef.close({script: this.scriptControl.value});
  }

  onCancel(): void {
    this.dialogRef.close();
  }

  onRevert(): void {
    if (!this.canRevert) {
      return;
    }

    this.scriptControl.reset(this.data.script);
  }
}
