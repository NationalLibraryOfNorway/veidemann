import {Component, Input} from '@angular/core';
import {ConfigObject} from '../../../../../shared/models/config';
import {DurationFormatPipe} from '../../../../../shared/pipes/duration-format.pipe';
import {NgxFilesizeModule} from 'ngx-filesize';
import {ScriptAnnotationsPipe} from '../../../pipe';
import {AsyncPipe} from '@angular/common';
import {ScriptAnnotationComponent} from '../../annotation/script-annotation/script-annotation.component';
import {MatLabel} from '@angular/material/form-field';

@Component({
  selector: 'app-crawljob-preview',
  templateUrl: './crawljob-preview.component.html',
  styleUrls: ['./crawljob-preview.component.css'],
  imports: [
    AsyncPipe,
    DurationFormatPipe,
    MatLabel,
    NgxFilesizeModule,
    ScriptAnnotationComponent,
    ScriptAnnotationsPipe
  ],
  standalone: true
})
export class CrawljobPreviewComponent {

  @Input()
  configObject: ConfigObject;

  constructor() {
  }
}
