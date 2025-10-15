import {Component, Input} from '@angular/core';
import {ConfigObject, RotationPolicy, SubCollectionType} from '../../../../../shared/models/config';
import {MatCheckbox} from '@angular/material/checkbox';
import {NgxFilesizeModule} from 'ngx-filesize';
import {MatLabel} from '@angular/material/form-field';
import {FormsModule} from '@angular/forms';

@Component({
  selector: 'app-collection-preview',
  templateUrl: './collection-preview.component.html',
  styleUrls: ['./collection-preview.component.css'],
  imports: [
    FormsModule,
    MatCheckbox,
    MatLabel,
    NgxFilesizeModule
  ],
  standalone: true
})
export class CollectionPreviewComponent {
  readonly RotationPolicy = RotationPolicy;
  readonly SubCollectionType = SubCollectionType;

  @Input()
  configObject: ConfigObject;

  constructor() {
  }
}
