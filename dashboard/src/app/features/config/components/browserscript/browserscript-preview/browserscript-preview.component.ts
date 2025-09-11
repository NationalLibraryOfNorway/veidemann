import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit} from '@angular/core';
import { BrowserScriptType, ConfigObject } from '../../../../../shared/models';
import { MatChipsModule } from '@angular/material/chips';
import {MatLabel} from '@angular/material/form-field';
import {EditorComponent} from 'ngx-monaco-editor-v2';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';
import {FormsModule} from '@angular/forms';


@Component({
  selector: 'app-browserscript-preview',
  templateUrl: './browserscript-preview.component.html',
  styleUrls: ['./browserscript-preview.component.css'],
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    EditorComponent,
    FlexLayoutModule,
    FormsModule,
    MatChipsModule,
    MatLabel
  ]
})
export class BrowserscriptPreviewComponent implements OnInit {
  readonly BrowserScriptType = BrowserScriptType;
  @Input()
  configObject: ConfigObject;

  language: string;

  editorOptions = {
    theme: window.matchMedia('(prefers-color-scheme: dark)').matches ? 'vs-dark' : 'vs',
    language: 'javascript',
    roundedSelection: true,
    readOnly: true,
    domReadOnly: true,
    contextmenu: false,
    minimap: {
      enabled: false
    }
  };

  constructor(protected cdr: ChangeDetectorRef) {
  }

  ngOnInit() {
    this.language = this.configObject.meta.name.split('.').slice(-1)[0];
  }

  initEditor(editor: any) {
    console.log('Editor initialized', editor);
    // editor.onDidChangeModelDecorations(() => {
    //   this.cdr.markForCheck();
    // });
  }



}
