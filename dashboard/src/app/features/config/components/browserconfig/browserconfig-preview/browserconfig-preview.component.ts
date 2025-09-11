import {Component, Input} from '@angular/core';
import {ConfigObject} from '../../../../../shared/models/config';
import {DurationFormatPipe} from '../../../../../shared/pipes/duration-format.pipe';
import {MatLabel} from '@angular/material/form-field';
import {MatChipsModule} from '@angular/material/chips';
import {ShortcutListComponent} from '../../shortcut/shortcut-list/shortcut-list.component';
import {FlexLayoutModule} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-browserconfig-preview',
  templateUrl: './browserconfig-preview.component.html',
  styleUrls: ['./browserconfig-preview.component.css'],
  imports: [
    DurationFormatPipe,
    FlexLayoutModule,
    MatChipsModule,
    MatLabel,
    ShortcutListComponent
  ],
  standalone: true
})
export class BrowserconfigPreviewComponent {
  @Input()
  configObject: ConfigObject;

  constructor() {
  }


}
