import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, inject } from '@angular/core';
import {BrowserScriptType, ConfigObject} from '../../../../../shared/models';
import {MatChipsModule} from '@angular/material/chips';
import {MatLabel} from '@angular/material/form-field';
import {EditorComponent} from 'ngx-monaco-editor-v2';
import {FlexDirective, FlexLayoutModule, LayoutDirective} from '@ngbracket/ngx-layout';
import {FormsModule} from '@angular/forms';
import type {editor} from 'monaco-editor';


@Component({
  selector: 'app-browserscript-preview',
  templateUrl: './browserscript-preview.component.html',
  styleUrls: ['./browserscript-preview.component.css'],
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    EditorComponent,
    FlexDirective,
    FlexLayoutModule,
    FormsModule,
    LayoutDirective,
    MatChipsModule,
    MatLabel
  ]
})

export class BrowserscriptPreviewComponent implements OnInit {
  protected cdr = inject(ChangeDetectorRef);

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

  ngOnInit() {
    this.language = this.configObject.meta.name.split('.').slice(-1)[0];
  }

  initEditor(editor: editor.IStandaloneCodeEditor) {
    setTimeout(() => {
      editor.layout();
    })
  }


}
